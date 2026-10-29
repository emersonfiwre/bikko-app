# Regras de negócio — Bikko

> **Leitura obrigatória para agentes e contribuidores.** Antes de alterar rotas, domínio, persistência, mocks, telas ou testes, leia este arquivo. O backend é a fonte de verdade: os apps só podem espelhar estados e regras que ele suporte.
>
> **Manutenção obrigatória:** qualquer mudança de regra de negócio deve atualizar este `RULES.md` no mesmo commit. Se uma regra necessária não estiver documentada, não a presuma: peça esclarecimento quando ela alterar o comportamento do produto; depois de definida, registre-a aqui junto com código, testes e mocks.

## 1. Papéis, escopo e segurança

- Há dois papéis autenticados: **Cliente** (Bikkofy) e **Prestador/Bikker** (Bikkerfy). Uma ação só pode ser executada pelo dono do recurso e pelo papel autorizado.
- A plataforma conecta as partes; o Bikko não presta o serviço, não cria vínculo trabalhista e **não recebe, processa, retém, transfere, garante ou estorna pagamentos**. Valor, meio, prazo, sinal, reembolso e divergências são combinados e liquidados diretamente entre Cliente e Prestador, fora da plataforma. Não apresentar checkout, carteira, escrow, "pagamento liberado" ou confirmação financeira pelo Bikko.
- Os únicos bypasses de autenticação permitidos são `11999990000` (Cliente) e `11988880000` (Prestador), estritamente em `ENV=development` com `LOCAL_TEST_ACCOUNTS=true`. Nunca ampliar essa lista, aceitar token mock arbitrário ou habilitar bypass em produção.
- CPF, telefone, endereço, localização e fotos são dados sensíveis/privados no contexto do produto: nunca registrar valores integrais em logs, expor dados de terceiros ou preencher dados fictícios em produção.

## 2. Solicitação, orçamento e ciclo de vida

1. O Cliente cria uma solicitação para um Prestador ativo, com descrição não vazia (máximo 280 caracteres), endereço/localização válidos e no máximo 10 fotos no total (sendo até 5 fotos na abertura da solicitação e até 5 adicionais caso o prestador solicite mais detalhes).
2. A solicitação nasce como `PENDING` no banco e é apresentada como `PENDING_BUDGET` nos contratos dos apps.
3. O Prestador pode enviar a proposta inicial (`BUDGET_RECEIVED`), pedir mais detalhes ou recusar/cancelar quando aplicável.
4. **Validade da proposta:** Toda proposta de orçamento tem prazo padrão de validade de 48 horas para resposta (aceite ou contraproposta). Expirado o prazo sem resposta, a solicitação é encerrada sem contratação.
5. Depois da proposta inicial, Cliente e Prestador podem alternar contrapropostas em `NEGOTIATING`.
6. O aceite usa a **última proposta persistida**; preço informado pelo cliente em uma chamada de aceite nunca substitui o valor aceito. O resultado é um pedido/contrato `ACTIVE`.
7. **Conclusão:** O Prestador conclui o atendimento ativo preenchendo o relatório técnico obrigatório (máximo 280 caracteres); o Cliente também pode confirmar a conclusão do atendimento pela interface. O resultado em ambos os casos é o estado terminal `COMPLETED`, que libera a avaliação mútua.
8. **Cancelamento e persistência interna:** Cliente ou Prestador podem cancelar conforme a transição permitida. No banco de dados (`quote_requests`), a recusa ou encerramento pelo Cliente é registrado internamente como `EXPIRED`, e a recusa pelo Prestador como `REJECTED`. Ambos os fluxos são normalizados nos contratos da API e dos aplicativos como o estado terminal unificado `CANCELLED`, preservando o motivo informado. Estados terminais (`COMPLETED` e `CANCELLED`) nunca reabrem.

Estados de apresentação canônicos: `PENDING_BUDGET`, `BUDGET_RECEIVED`, `NEGOTIATING`, `ACTIVE`, `COMPLETED` e `CANCELLED`. Compatibilidades como `PENDING`, `ACCEPTED`, `SCHEDULED` e `IN_PROGRESS` devem ser normalizadas para esses estados de produto.

## 3. Contrapropostas e pedido de detalhes

- São permitidas **até 5 rodadas de contraproposta**, além da proposta inicial. Cada contraproposta com preço válido consome uma rodada; a sexta deve retornar conflito, sem alterar histórico ou estado.
- A autoria e a ordem do histórico são imutáveis. Sempre exibir toda a sequência, incluindo proposta inicial, pedidos de detalhes, respostas com fotos e contrapropostas posteriores.
- **Pedido de mais detalhes** é uma mensagem sem valor/preço. Não consome rodada, não é contraproposta financeira e permite ao outro participante responder com mensagem e até 5 fotos complementares.
- Fotos de evidência/detalhes podem ser anexadas na criação da solicitação (até 5) e como resposta a um pedido explícito de mais detalhes (até 5). Não permitir que uma contraproposta financeira comum anexe novas fotos depois da primeira interação; isso evita crescimento indevido de anexos e mantém o fluxo auditável.
- O histórico e o contador de tentativas devem refletir apenas propostas financeiras. Pedido e resposta de detalhes aparecem no histórico, mas usam tentativa `0`/sem consumo de rodada.
- Um Prestador inativo não recebe novos leads, mas pedidos já atribuídos não podem ficar inacessíveis nem impedir a conclusão/cancelamento regular.

## 4. Avaliações

- Avaliações são permitidas somente quando o pedido está `COMPLETED`; pedido cancelado nunca oferece nem aceita avaliação.
- O Cliente avalia o Prestador/serviço uma vez por pedido. O Prestador avalia o Cliente uma vez por pedido e pode incluir relatório técnico.
- **Imutabilidade e irreversibilidade:** A avaliação é única e definitiva. Não há endpoint ou permissão para edição ou exclusão de avaliação após o envio. Na interface dos apps, o envio com sucesso desabilita permanentemente a ação de avaliar.
- Nota é obrigatória de 1 a 5. Comentário do Cliente tem máximo de 1000 caracteres e até 10 tags; comentário da avaliação do Cliente pelo Prestador tem máximo de 120 caracteres e relatório técnico máximo de 280.
- A avaliação atualiza média/contagem do destinatário, nunca do autor. Textos e telas devem identificar corretamente quem avaliou quem.
- Duplicidade natural (`pedido + autor`) é bloqueada. A repetição da mesma chave de idempotência e do mesmo payload deve retornar sucesso e a avaliação originalmente criada; mesma chave com payload diferente retorna conflito.

## 5. Portfólio de serviços do Prestador

- **Obrigatoriedade de Categoria Oficial:** O Prestador só pode criar ou editar um serviço vinculando-o obrigatoriamente a uma das categorias oficiais registradas no backend (`GET /categories`). A interface bloqueia o salvamento até a seleção válida de uma categoria oficial.
- **Limites do serviço:** Título com até 255 caracteres, descrição com até 4000 caracteres e no máximo 5 fotos de galeria na pasta `services`.
- **Inativação vs Exclusão (Soft Delete):** A remoção de um serviço pelo Prestador realiza exclusão lógica (*soft delete* / `is_active = false`) caso existam solicitações ou contratos históricos vinculados, garantindo a integridade dos registros de clientes. Se não houver vínculos, a exclusão física é permitida.

## 6. Identidade, perfis e LGPD

- **Telefone da conta mestre:** O número validado via SMS OTP é a chave mestre de identidade do usuário. Ele nunca é alterado por campos informados em formulários de solicitação de serviço.
- **Telefone de contato transitório:** O telefone editável no formulário de orçamento (`RequestQuoteScreen`) é restrito àquela solicitação específica e tratado pela aplicação como contato transitório não verificado.
- **Conflito de e-mail (HTTP 409):** O e-mail no perfil é opcional na atualização, mas se informado e já pertencer a outro usuário ativo no sistema, o backend rejeita a operação com `409 Conflict` (`domain.ErrConflict`), sem sobrescrever dados de terceiros.
- **Exclusão de conta (Soft Delete):** A exclusão solicitada pelo usuário marca `deleted_at = NOW()` e inativa imediatamente todos os serviços do prestador (`is_active = false`). Usuários com `deleted_at` preenchido são impedidos de fazer login e não aparecem em buscas ou radares.

## 7. Idempotência e falhas de rede

- Criação de solicitação, avaliações e mutações de portfólio (criação, edição e remoção de serviços) exigem `Idempotency-Key` não vazio, com até 128 caracteres. Toda mutação de status, proposta, contraproposta, aceite, cancelamento e conclusão também deve receber uma chave estável do cliente.
- A chave é vinculada ao ator e ao fingerprint do payload. Mesmo ator + mesma chave + mesmo conteúdo = replay seguro; mesma chave + conteúdo diferente = `409`/conflito; chave nova para operação já concluída respeita a regra de duplicidade do domínio.
- Repetições nunca podem criar serviço, proposta, pedido, avaliação, notificação ou atualização de reputação adicional.
- Nos aplicativos Android, `PendingMutationKeys` armazena a chave de forma persistente (`SharedPreferences`) para sobreviver a process death até a confirmação de sucesso pelo backend. Em caso de logout/limpeza de sessão, as chaves pendentes são expurgadas para isolamento entre contas.
- Use transações, locks e índices de unicidade como garantia. Não substitua essas proteções por memória de processo, mocks ou flags de UI.

## 8. Fotos, perfil e armazenamento

- Foto de perfil só é atualizada depois de upload real bem-sucedido e persistência do URL pelo endpoint de perfil (`PUT /profile/photo`). Nunca atualizar a UI como sucesso antecipado.
- Use pastas tipadas: `profiles` para avatar, `services` para portfólio e `quotes` para evidências/solicitações.
- Em desenvolvimento, `LOCAL_STORAGE=true` é armazenamento local real; em produção, R2 é obrigatório. Não simular URL de upload bem-sucedido sem persistir o arquivo.
- `bikker-maoson.webp` é apenas hint visual do editor, nunca a imagem de capa automática de um serviço. Serviço sem foto deve mostrar estado de sem imagem.

## 9. Implementação, observability e testes

- Falhas HTTP, de upload ou de domínio devem resultar em estado de erro/retry visível; nunca em sucesso local fictício. Propague `CancellationException` e registre contexto de cancelamento/erro no observability.
- Mocks devem seguir o contrato HTTP e os estados do backend, ter dados coerentes e não mascarar erros de API.
- Preserve UDF nos apps e valide permissões/estado no backend. Não confiar em `user_id`, preço, papel, status ou URL vindos do corpo da requisição.
- Para toda alteração de regra, atualizar testes de unidade/integração, mocks relevantes e este arquivo. Nenhuma migration já aplicada pode ser alterada; crie uma nova migration imutável.
