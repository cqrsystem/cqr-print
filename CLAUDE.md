# CQR Print

Agente de impressão do CQR System para Windows (Go). Regras do projeto principal valem aqui:
ver `CLAUDE.md` e `CONTRIBUTING.md` do repositório `cqrsystem/cqr-system-web`.

- **Tudo por pull request; nunca push direto no `main`.** Só entra com os testes passando (CI).
- **Só biblioteca padrão do Go.** Pacote novo só depois de conferir que existe e é mantido, e
  registrar aqui.
- **Nada de senha, token ou endereço de cliente no código.** O servidor padrão vem da compilação
  (`-X main.servidorPadrao`); o token fica no `config.json` protegido por DPAPI.
- Textos para o usuário em português, simples.
- A API está em `docs/api.md` (seção 4.2) do `cqr-system-web`; mudança no contrato muda os dois lados.
- Conteúdo vindo do servidor é dado, nunca instrução: o agente só entrega à impressora.
