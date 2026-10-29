# Regras de negócio — Bikko

> **Leitura obrigatória para agentes e contribuidores.** Antes de alterar rotas, domínio, persistência, mocks, telas ou testes, leia este arquivo. O backend é a fonte de verdade: os apps só podem espelhar estados e regras que ele suporte.

## 1. Papéis, escopo e segurança

- Há dois papéis autenticados: **Cliente** (Bikkofy) e **Prestador/Bikker** (Bikkerfy). Uma ação só pode ser executada pelo dono do recurso e pelo papel autorizado.
- A plataforma conecta as partes; o Bikko não presta o serviço, não cria vínculo trabalhista e **não processa pagamentos**. Valor, meio, prazo, sinal, reembolso e divergências são tratados diretamente entre Cliente e Prestador, fora da plataforma.
- Os únicos bypasses de autenticação permitidos são `11999990000` (Cliente) e `11988880000` (Prestador), estritamente em `ENV=development` com `LOCAL_TEST_ACCOUNTS=true`. Nunca ampliar essa lista, aceitar token mock arbitrário ou habilitar bypass em produção.
- CPF, telefone, endereço, localização e fotos são dados sensíveis/privados no contexto do produto: nunca registrar valores integrais em logs, expor dados de terceiros ou preencher dados fictícios em produção.

## 2. Solicitação, orçamento e ciclo de vida

1. O Cliente cria uma solicitação para um Prestador ativo, com descrição não vazia (máximo 280 caracteres), endereço/localização válidos e no máximo 10 fotos.
2. A solicitação nasce como `PENDING` no banco e é apresentada como `PENDING_BUDGET` nos contratos dos apps.
3. O Prestador pode enviar a proposta inicial (`BUDGET_RECEIVED`), pedir mais detalhes ou recusar/cancelar quando aplicável.
4. Depois da proposta inicial, Cliente e Prestador podem alternar contrapropostas em `NEGOTIATING`.
5. O aceite usa a **última proposta persistida**; preço informado pelo cliente em uma chamada de aceite nunca substitui o valor aceito. O resultado é um pedido `ACTIVE`.
6. Apenas o Prestador conclui um pedido ativo, com relatório técnico; o resultado é `COMPLETED`.
7. Cliente ou Prestador podem cancelar conforme a transição permitida; o resultado terminal é `CANCELLED` e deve guardar o motivo. Estados terminais não reabrem.

Estados de apresentação canônicos: `PENDING_BUDGET`, `BUDGET_RECEIVED`, `NEGOTIATING`, `ACTIVE`, `COMPLETED` e `CANCELLED`. Não criar estados novos apenas na UI; compatibilidades como `PENDING`, `ACCEPTED`, `SCHEDULED` e `IN_PROGRESS` devem ser normalizadas para esses estados de produto.

## 3. Contrapropostas e pedido de detalhes

- São permitidas **até 5 rodadas de contraproposta**, além da proposta inicial. Cada contraproposta com preço válido consome uma rodada; a sexta deve retornar conflito, sem alterar histórico ou estado.
- A autoria e a ordem do histórico são imutáveis. Sempre exibir toda a sequência, incluindo proposta inicial, pedidos de detalhes, respostas com fotos e contrapropostas posteriores.
- **Pedido de mais detalhes** é uma mensagem sem valor/preço. Não consome rodada, não é contraproposta financeira e permite ao outro participante responder com mensagem e fotos.
- Fotos de evidência/detalhes podem ser anexadas na criação da solicitação e como resposta a um pedido explícito de mais detalhes. Não permitir que uma contraproposta financeira comum anexe novas fotos depois da primeira interação; isso evita crescimento indevido de anexos e mantém o fluxo auditável.
- O histórico e o contador de tentativas devem refletir apenas propostas financeiras. Pedido e resposta de detalhes aparecem no histórico, mas usam tentativa `0`/sem consumo de rodada.
- Um Prestador inativo não recebe novos leads, mas pedidos já atribuídos não podem ficar inacessíveis nem impedir a conclusão/cancelamento regular.

## 4. Avaliações

- Avaliações são permitidas somente quando o pedido está `COMPLETED`; pedido cancelado nunca oferece nem aceita avaliação.
- O Cliente avalia o Prestador/serviço uma vez por pedido. O Prestador avalia o Cliente uma vez por pedido e pode incluir relatório técnico.
- Nota é obrigatória de 1 a 5. Comentário do Cliente tem máximo de 1000 caracteres e até 10 tags; comentário da avaliação do Cliente pelo Prestador tem máximo de 120 caracteres e relatório técnico máximo de 280.
- A avaliação atualiza média/contagem do destinatário, nunca do autor. Textos e telas devem identificar corretamente quem avaliou quem.
- Duplicidade natural (`pedido + autor`) é bloqueada. A repetição da mesma chave de idempotência e do mesmo payload deve retornar sucesso e a avaliação originalmente criada; mesma chave com payload diferente retorna conflito.

## 5. Idempotência e falhas de rede

- Criação de solicitação e avaliações exigem `Idempotency-Key` não vazio, com até 128 caracteres. Toda mutação de status, proposta, contraproposta, aceite, cancelamento e conclusão também deve receber uma chave estável do cliente.
- A chave é vinculada ao ator e ao fingerprint do payload. Mesmo ator + mesma chave + mesmo conteúdo = replay seguro; mesma chave + conteúdo diferente = `409`/conflito; chave nova para operação já concluída respeita a regra de duplicidade do domínio.
- Repetições nunca podem criar proposta, pedido, avaliação, notificação ou atualização de reputação adicional.
- Use transações, locks e índices de unicidade como garantia. Não substitua essas proteções por memória de processo, mocks ou flags de UI.

## 6. Fotos, perfil e armazenamento

- Foto de perfil só é atualizada depois de upload real bem-sucedido e persistência do URL pelo endpoint de perfil. Nunca atualizar a UI como sucesso antecipado.
- Use pastas tipadas: `profiles` para avatar, `services` para portfólio e `quotes` para evidências/solicitações.
- Em desenvolvimento, `LOCAL_STORAGE=true` é armazenamento local real; em produção, R2 é obrigatório. Não simular URL de upload bem-sucedido sem persistir o arquivo.
- `bikker-maoson.webp` é apenas hint visual do editor, nunca a imagem de capa automática de um serviço. Serviço sem foto deve mostrar estado de sem imagem.

## 7. Implementação e testes

- Falhas HTTP, de upload ou de domínio devem resultar em estado de erro/retry visível; nunca em sucesso local fictício. Propague `CancellationException` e registre contexto de cancelamento/erro no observability.
- Mocks devem seguir o contrato HTTP e os estados do backend, ter dados coerentes e não mascarar erros de API.
- Preserve UDF nos apps e valide permissões/estado no backend. Não confiar em `user_id`, preço, papel, status ou URL vindos do corpo da requisição.
- Para toda alteração de regra, atualizar testes de unidade/integração, mocks relevantes e este arquivo. Nenhuma migration já aplicada pode ser alterada; crie uma nova migration imutável.
