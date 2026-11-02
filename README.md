# CQR Print

Agente de impressão de etiquetas do **CQR System** para **Windows**. Fica num computador da
fábrica, busca as etiquetas no servidor do CQR System e entrega às impressoras:

- **Zebra** (ZPL): direto pelo spooler do Windows (USB ou rede) ou pelo IP, porta 9100;
- **Argox, laser, jato de tinta**: a etiqueta pronta como imagem, pelo driver do Windows, no
  tamanho da etiqueta.

Assim dá para imprimir de **qualquer aparelho** (Desk, tablet do posto, celular): o pedido vai ao
servidor e o CQR Print imprime. Só o agente chama o servidor (HTTPS); nada precisa ser aberto no
firewall da fábrica.

Proposta e decisões: `docs/propostas/agente-de-impressao.md`; API: `docs/api.md`, seção 4.2
(ambos no repositório `cqr-system-web`).

## Instalar (no computador da fábrica)

1. Baixe o `cqrprint.exe` (última versão em *Releases*).
2. Clique com o botão direito → **Executar como administrador**.
3. Responda: endereço do CQR System, **CNPJ**, **matrícula** e **senha** de um Supervisor ou
   Administrador, e o nome do computador.
4. Pronto: o programa fica em `C:\Program Files\CQR Print`, roda como serviço (tarefa agendada do
   Windows, conta SYSTEM) e volta sozinho quando o computador é ligado.
5. No **Desk → Configurações → Impressoras**, cadastre as impressoras deste computador (a lista do
   Windows aparece lá) e use **Imprimir teste**.

O computador aparece no Desk em **Configurações → Aparelhos** como "Agente de impressão"; revogar
ali desliga o agente.

**Atenção:** como roda como SYSTEM, o CQR Print enxerga as impressoras **instaladas no computador**
(para todos os usuários). Impressora conectada só no perfil de um usuário pode não aparecer.
Para impressoras comuns (Argox pelo driver, laser), configure no driver o **tamanho do papel** igual
ao da etiqueta.

## Comandos

```
cqrprint                      instala (dois cliques, como administrador)
cqrprint status               empresa, servidor e impressoras configuradas
cqrprint impressoras          impressoras que o Windows enxerga
cqrprint testar "ZD220"       etiqueta ZPL de teste na impressora do Windows
cqrprint testar --ip 10.0.0.5 etiqueta ZPL de teste pelo IP (porta 9100)
cqrprint ativar               ativa de novo (outro CNPJ ou agente revogado)
cqrprint desinstalar          remove a tarefa e a configuração
cqrprint executar             roda na janela (o que a tarefa agendada faz)
```

Registro: `C:\ProgramData\CQR Print\cqrprint.log`. A configuração fica em
`C:\ProgramData\CQR Print\config.json`, com o token protegido pelo Windows (DPAPI da máquina: o
arquivo não funciona em outro computador).

## Desenvolvimento

- Go 1.24, **só biblioteca padrão** (chamadas do Windows por `syscall`: winspool, gdi32, crypt32).
- `go test ./...` roda em qualquer sistema (as partes do Windows têm versão vazia fora dele).
- `scripts/compilar.sh 1.0.0 https://servidor` gera `dist/cqrprint.exe`.
- Versão: criar a tag `v1.0.0` publica o `.exe` em *Releases* (GitHub Actions). O servidor padrão
  da instalação vem da variável `CQR_SERVIDOR_PADRAO` do repositório (nada fixo no código).
- **Falta:** assinatura digital do `.exe` (certificado, antes de ir para os clientes),
  atualização automática, teste nas impressoras de verdade (Zebra e Argox).
