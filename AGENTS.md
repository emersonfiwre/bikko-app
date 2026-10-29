# Instruções para agentes

1. Leia `GEMINI.md` e `RULES.md` integralmente antes de planejar ou alterar código, banco, mocks, testes, rotas ou documentação.
2. `RULES.md` é o contrato de negócio canônico. O backend é a fonte de verdade dos estados, permissões, transições e idempotência.
3. Ao modificar uma regra de negócio, ou implementar uma regra que ainda não esteja documentada, atualize `RULES.md` no mesmo commit. Se a regra necessária estiver ambígua e puder alterar o comportamento do produto, não a presuma: peça esclarecimento ao usuário antes de implementar.
4. Atualize também testes, mocks e documentação afetados. Não introduza sucesso local falso, bypasses novos ou estados que o contrato não suporte.

Estas instruções complementam `GEMINI.md`; em caso de conflito, siga as instruções mais específicas e a autorização atual do usuário.
