# Raport analizy implementacji pola Oersteda w amumax

Data analizy: 2026-02-23
Zakres: `src/engine/oersted.go`, `src/mag/oerstedkernel.go`, `src/cuda/conv_oersted.go`, `src/cuda/oerstedkernmul3d.cu` + zaleznosci (`average`, `excitation`, energia).

## Podsumowanie

Implementacja **nie jest w pelni poprawna** pod wzgledem fizycznym i numerycznym.

Co jest dobre:
- Matematyka kernela Biota-Savarta i mnozenie wektorowe w przestrzeni Fouriera sa poprawne.
- Skalowanie FFT (`1/N`) jest ustawione poprawnie.
- Znaki i orientacja pola przechodza sanity-checki numeryczne.

Co jest problematyczne:
- Istnieja istotne bledy/ryzyka w **invalidacji cache** (mozliwe stale, fizycznie bledne pole).
- `I_oersted` jest traktowane jako srednia przestrzenna, co jest semantycznie i wydajnosciowo slabe.
- Wklad Oersteda jest dodawany do `B_eff`, ale nie do energii (`E_total`, `E_Zeeman`), co daje niespojnosc.
- Prad jest wymuszony przez maske geometrii magnetycznej.
- Brak testow regresyjnych dedykowanych Oerstedowi.

## 1. Ocena fizyczna

### 1.1 Co jest fizycznie poprawne

1. Kernel:
- `K(r) = mu0/(4pi) * dV * r / |r|^3` w `src/mag/oerstedkernel.go:15`, `src/mag/oerstedkernel.go:59`, `src/mag/oerstedkernel.go:82`.
- Jednostki sa spojne i daja wynik w teslach.

2. Cross-product:
- `B = J x K` jest spojne z prawem Biota-Savarta (`src/mag/oerstedkernel.go:20`, `src/cuda/oerstedkernmul3d.cu:48-58`).

3. Kierunek pola:
- Dla pradu w `+y` pole po stronie `+x` ma znak `-z` (zgodnie z regula prawej dloni) w niezaleznym sanity-checku.

### 1.2 Ograniczenia fizyczne / niespojnosci

1. Prad jest maskowany geometria magnetyczna:
- `recomputeOerstedBase` przekazuje `vol := Geometry.Gpu()` (`src/engine/oersted.go:163`).
- `copyPadVol` mnozy zrodlo przez `vol` (`src/cuda/conv_oersted.go:73`, `src/cuda/conv_oersted.go:162-178`, `src/cuda/copypad.cu:17-18`).

Skutek:
- Nie da sie naturalnie modelowac zrodla pradu poza obszarem geometrii magnetycznej (np. osobny przewodnik/stripline), chyba ze uzytkownik sztucznie rozszerzy geometrie.

2. Brak wkladu do energii:
- Oersted jest dodawany do `B_eff` (`src/engine/effectivefield.go:18`),
- ale energia liczona jest z zarejestrowanych termow (`src/engine/energy.go:21-24`), a Zeeman opiera sie tylko o `B_ext` (`src/engine/zeeman.go:9`, `src/engine/zeeman.go:16`).

Skutek:
- `E_total` i `E_Zeeman` nie zawieraja czesci od `B_oersted`.

## 2. Ocena numeryczna

### 2.1 Co jest numerycznie poprawne

1. Pipeline FFT:
- FW FFT J, mnozenie wektorowe w k-przestrzeni, BW FFT (`src/cuda/conv_oersted.go:51-65`).

2. Normalizacja:
- Kernel skalowany przez `1/InputLen` (`src/cuda/conv_oersted.go:111`, `src/cuda/conv_oersted.go:129-131`).

3. Weryfikacja niezalezna:
- Test FFT-vs-direct (losowe 3D J): blad wzgledny ~`4.75e-16`.
- Test dlugiego przewodnika: zgodnosc amplitudy z analityka (`~1.0e-8` relatywnie, praktycznie pelna zgodnosc).

### 2.2 Problemy numeryczne / implementacyjne (istotne)

1. Niepelna invalidacja cache pola bazowego (WYSOKIE RYZYKO)

Warunek przebudowy cache:
- tylko mesh + `JOersted.Revision()` (`src/engine/oersted.go:112-117`).

Ale `B_base` zalezy tez od:
- geometrii `vol` (`src/engine/oersted.go:163`, `src/cuda/copypad.cu:17`),
- mapy regionow (przez `RegionAddV` w `JOersted.Slice()`, `src/engine/excitation.go:46`),
- czasu, jesli `J_oersted` ma skladowe czasowe.

`JOersted.Revision()` rosnie tylko przy jawnej mutacji obiektu (`src/engine/excitation.go:114-126`), natomiast aktualizacja funkcji czasu dzieje sie w `regionwise.update()` bez zmiany rev (`src/engine/parameter.go:100-115`).

Skutek:
- Pole moze pozostac stale mimo realnej zmiany `J_oersted`, geometrii albo regionow.
- To jest krytyczne dla wiarygodnosci wynikow przy dynamicznych scenariuszach.

2. `I_oersted` liczone jako srednia przestrzenna (SREDNIE/WYSOKIE RYZYKO semantyczne)

- Amplituda: `amp := float32(IOersted.average())` (`src/engine/oersted.go:70`, `src/engine/oersted.go:94`).
- `average()` dla `scalarExcitation` to pelna srednia po siatce (`src/engine/scalar_excitation.go:117`, `src/engine/average.go:11-23`).

Skutki:
- Jesli `I_oersted` nie jest jednorodne przestrzennie, model redukuje je do jednej liczby (sredniej), co moze byc fizycznie niepoprawne i zalezne od rozmiaru siatki.
- Dodatkowy koszt O(N) na kazdy odczyt amplitudy oslabia zalozenie "cheap scaling per step".

3. Aproksymacja punktowa kernela (SREDNIE RYZYKO dokladnosci bliskiego pola)

- Kernel korzysta z aproksymacji `dV` w centrum komorki (`src/mag/oerstedkernel.go:59-60`, `src/mag/oerstedkernel.go:82-88`) i `self-term = 0` (`src/mag/oerstedkernel.go:75-77`).
- Brak analogicznej do demaga integracji wysokiej dokladnosci po objetosci.

Skutek:
- Dla grubych komorek i bliskiego pola bledy lokalne moga byc istotne.

## 3. Klasyfikacja ryzyka

1. Wysokie:
- Niepelna invalidacja cache `B_base` (czas/geometria/regiony).

2. Srednie:
- Brak termu energetycznego Oersteda.
- Semantyka `I_oersted` jako sredniej przestrzennej.
- Wymuszenie geometrii magnetycznej jako maski zrodla pradu.
- Brak testow Oersted.

3. Niskie/Swiadome ograniczenia:
- Brak wsparcia PBC (`src/mag/oerstedkernel.go:30-34`).

## 4. Rekomendacje

1. Cache/inwalidacja:
- Rozszerzyc sygnature cache co najmniej o rewizje geometrii i regionow.
- Dodac tryb: jesli `J_oersted` jest czasowo zalezne, wymusic przebudowe `B_base` (albo jawnie tego zabronic z bledem/warningiem).

2. `I_oersted`:
- Ograniczyc do globalnego skalaru czasu (bez przestrzeni), albo zmienic API i dokumentacje.
- Uniknac liczenia przez srednia po calym meshu, jesli ma byc "global amplitude".

3. Energia:
- Dodac `Edens_oersted` / `E_oersted` i zarejestrowac do `E_total`, albo wyraznie udokumentowac, ze Oersted nie wchodzi do energetyki.

4. Maska zrodla:
- Rozwazyc osobna maske przewodnika (niezalezna od `Geometry`) lub opcje disable maskowania.

5. Testy:
- Dodac testy regresyjne:
  - FFT vs direct convolution na malej siatce.
  - test znaku i skali dla prostego przewodnika.
  - test invalidacji cache po zmianie geometrii/regionow/czasowo-zaleznego `J_oersted`.

## 5. Informacje o weryfikacji wykonania

Wykonane sanity-checki:
- Niezalezny test numeryczny FFT-vs-direct: zgodnosc do precyzji maszynowej.
- Niezalezny test analityczny (dlugi przewodnik): bardzo dobra zgodnosc amplitudy i znaku.

Ograniczenie srodowiska:
- Nie bylo mozliwe uruchomienie pelnej kompilacji/testow `engine` z powodu braku naglowkow CUDA (`cuda.h`, `curand.h`) w srodowisku.
