#!/usr/bin/env bash
# Gera cmd/cqrprint/recursos_windows_amd64.syso: nome, empresa, versão e descrição dentro do .exe
# (o Windows mostra em Propriedades → Detalhes) e o manifesto que pede administrador ao abrir.
# Programa sem essas informações chama mais a atenção dos antivírus.
# Precisa do windres (Ubuntu: apt install binutils-mingw-w64-x86-64).
# Uso: windows/recursos.sh 1.2.3
set -euo pipefail
cd "$(dirname "$0")/.."
VERSAO="${1:-0.0.0}"
# Versão numérica do Windows: 1.2.3 -> 1,2,3,0 (o que não for número vira 0).
IFS=. read -r MA MI PA _ <<<"${VERSAO%%[-+]*}.0.0.0"
num() { [[ "$1" =~ ^[0-9]+$ ]] && echo "$1" || echo 0; }
MA=$(num "$MA"); MI=$(num "$MI"); PA=$(num "$PA")
ANO=$(date +%Y)
cat > windows/cqrprint.rc <<RC
1 24 "cqrprint.manifest"
1 VERSIONINFO
FILEVERSION ${MA},${MI},${PA},0
PRODUCTVERSION ${MA},${MI},${PA},0
FILEOS 0x40004
FILETYPE 0x1
BEGIN
  BLOCK "StringFileInfo"
  BEGIN
    BLOCK "041604B0"
    BEGIN
      VALUE "CompanyName", "CQR System"
      VALUE "FileDescription", "CQR Print - impressão de etiquetas do CQR System"
      VALUE "FileVersion", "${VERSAO}"
      VALUE "InternalName", "cqrprint"
      VALUE "LegalCopyright", "© ${ANO} CQR System"
      VALUE "OriginalFilename", "cqrprint.exe"
      VALUE "ProductName", "CQR Print"
      VALUE "ProductVersion", "${VERSAO}"
      VALUE "Comments", "Código aberto: https://github.com/cqrsystem/cqr-print"
    END
  END
  BLOCK "VarFileInfo"
  BEGIN
    VALUE "Translation", 0x0416, 1200
  END
END
RC
(cd windows && x86_64-w64-mingw32-windres --preprocessor=./sem-preprocessador.sh -c 65001 -O coff -o ../cmd/cqrprint/recursos_windows_amd64.syso cqrprint.rc)
rm windows/cqrprint.rc
echo "cmd/cqrprint/recursos_windows_amd64.syso ($VERSAO)"
