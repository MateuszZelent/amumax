repo_dir := justfile_directory()
podman := "podman --cgroup-manager=cgroupfs"
go_cache_env := "-e GOPATH=/tmp/go -e GOCACHE=/tmp/go-cache -e GOMODCACHE=/tmp/go/pkg/mod"

run-dev:
	{{podman}} run -it --rm {{go_cache_env}} -p 35367:35367 -v {{repo_dir}}:/src \
	--device=nvidia.com/gpu=all \
	matmoa/amumax:build bash

image:
	{{podman}} build -t matmoa/amumax:build {{repo_dir}}

build-cuda: 
	{{podman}} run --rm {{go_cache_env}} -v {{repo_dir}}:/src matmoa/amumax:build sh src/cuda/build_cuda.sh

copy-pcss:
	scp -r ./build/amumax pcss:grant_398/scratch/bin/amumax_versions/amumax$(date -I)
	ssh pcss "cd ~/grant_398/scratch/bin && ln -sf amumax_versions/amumax$(date -I) amumax"

build-frontend: 
	{{podman}} run --rm \
		-v {{repo_dir}}:/src \
		-w /src/frontend \
		docker.io/node:18.20.4-alpine3.20 \
		sh -lc 'npm run build && rm -rf /src/src/api/static && mv dist /src/src/api/static'

build:
	{{podman}} run --rm {{go_cache_env}} -v {{repo_dir}}:/src matmoa/amumax:build bash -lc 'if [ -d build ]; then if touch build/.codex-write-test 2>/dev/null; then rm -f build/.codex-write-test && rm -rf build; else mv build build.stale.$(date +%s); fi; fi && mkdir -p build && go build -v -ldflags "-X github.com/MathieuMoalic/amumax/src/version.VERSION=$(date -u +%Y.%m.%d)" -o build/amumax'

update-flake-gh-hash VERSION:
	#!/usr/bin/env sh
	set -euxo pipefail
	sed -i 's/releaseVersion = "[^"]*"/releaseVersion = "'"{{VERSION}}"'"/' flake.nix

	GH_HASH=$(nix-prefetch-github MathieuMoalic amumax --rev {{VERSION}} | jq -r '.hash')
	escaped_hash=$(printf '%s' "$GH_HASH" | sed 's/[&/\]/\\&/g')
	sed -i "s/hash = pkgs.lib.fakeHash;/hash = \"$escaped_hash\";/" flake.nix

test:
	{{podman}} run --rm {{go_cache_env}} -v {{repo_dir}}:/src matmoa/amumax:build bash -lc 'packages=$(go list ./src/... | grep -Ev "^github.com/MathieuMoalic/amumax/src/cuda($|/)"); go test $packages'

test-cuda:
	{{podman}} run --rm {{go_cache_env}} --device=nvidia.com/gpu=all -v {{repo_dir}}:/src matmoa/amumax:build go test ./src/cuda/...
	
release: 
	#!/usr/bin/env sh
	set -euxo pipefail
	git checkout main

		if [ -n "$(git status --porcelain)" ]; then
		echo "Working directory is not clean. Please commit or stash your changes."
		exit 1
	fi
	
	git pull
	VERSION=$(date -u +'%Y.%m.%d')
	gh release view $VERSION &>/dev/null && gh release delete $VERSION -y
	git show-ref --tags $VERSION &>/dev/null && git tag -d $VERSION && git push --tags

	just image build-cuda build-frontend build

	# We need to commit before the release
	git add .
	if git diff-index --quiet HEAD --; then
		echo "No changes to commit. Skipping commit step."
	else
		git commit -m "Release of $VERSION"
	fi
	git push
	gh release create $VERSION ./build/* --title $VERSION --notes "Release of ${VERSION}"
	just copy-pcss
	just flake-release

flake-release:
	#!/usr/bin/env sh
	set -euxo pipefail
	VERSION=$(date -u +'%Y.%m.%d')
	just update-flake-hashes-git
	just update-flake-gh-hash ${VERSION}
	nix run . -- -v
	git add .
	git commit -m "Update github hash for the release of ${VERSION}"
	git push

update-flake-hashes-git:
	#!/usr/bin/env sh
	set -euxo pipefail

	echo "Resetting npmDepsHash and vendorHash to placeholder values..."
	sed -i 's/npmDepsHash = "sha256-[^\"]*";/npmDepsHash = pkgs.lib.fakeHash;/' flake.nix
	sed -i 's/vendorHash = "sha256-[^\"]*";/vendorHash = pkgs.lib.fakeHash;/' flake.nix
	sed -i 's/hash = "sha256-[^\"]*";/hash = pkgs.lib.fakeHash;/' flake.nix

	echo "Starting the hash update process..."

	update_hashes() {
		echo "Running nix command to capture output and find new hashes..."
		output=$(nix run .#git -- -v 2>&1 || true)

		new_hash=$(echo "$output" | grep 'got:' | awk '{print $2}')
		escaped_hash=$(printf '%s' "$new_hash" | sed 's/[&/\]/\\&/g')

		if [[ -n "$new_hash" ]]; then
			echo "New hash found: $new_hash"
			if [[ "$output" == *"frontend-git-npm-deps.drv':"* ]]; then
				echo "Updating npmDepsHash in flake.nix..."
				sed -i "s/npmDepsHash = pkgs.lib.fakeHash;/npmDepsHash = \"$escaped_hash\";/" flake.nix
			elif [[ "$output" == *"git-go-modules.drv':"* ]]; then
				echo "Updating vendorHash in flake.nix..."
				sed -i "s/vendorHash = pkgs.lib.fakeHash;/vendorHash = \"$escaped_hash\";/" flake.nix
			else
				echo "Error: None of the expected patterns found in the output." >&2
				return 1
			fi
		else
			echo "Error: No new hash found in the output." >&2
			return 1
		fi
	}

	echo "Updating hashes..."
	update_hashes
	echo "First update completed. Running the update again..."
	update_hashes

	echo "Running final test to verify updated hashes..."
	nix run .#git -- -v

	echo "Hash update process completed."
