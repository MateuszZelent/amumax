#!/bin/sh
set -e

# Check for necessary commands
for cmd in curl tar xz; do
  if ! command -v $cmd > /dev/null 2>&1; then
    echo "Error: '$cmd' is not installed. Please install it and try again." >&2
    exit 1
  fi
done

DEST=$1

# Prompt for installation path if not provided
if [ -z "$DEST" ]; then
  printf "Where to install amumax? [Default=$HOME/.local/bin]: "
  read DEST
  if [ -z "$DEST" ]; then
    DEST="$HOME/.local/bin"
  fi
fi

mkdir -p "$DEST"
DEST=$(realpath "$DEST")

# Warn if DEST is not in PATH
case ":$PATH:" in
  *:"$DEST":*) ;;
  *) 
    echo && echo " !!! WARNING !!! '$DEST' not in PATH!"
    echo "Consider adding '$DEST' to your PATH." >&2
    ;;
esac

# Download and install amumax
cd $DEST
echo "Downloading amumax from GitHub..."
curl -Ls https://github.com/mathieumoalic/amumax/releases/latest/download/amumax -o amumax

# Download necessary libraries
echo "Downloading libcufft.so.11..."
curl -Ls https://github.com/mathieumoalic/amumax/releases/latest/download/libcufft.so.11 -o libcufft.so.11

echo "Downloading libcurand.so.10..."
curl -Ls https://github.com/mathieumoalic/amumax/releases/latest/download/libcurand.so.10 -o libcurand.so.10

# Make amumax executable
echo "Setting amumax as executable"
chmod +x amumax

# Completion message
echo "Installation complete. You can now use 'amumax'."
