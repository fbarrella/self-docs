# Project Requirements Document (PRD) - self-docs

## 1. Visão Geral do Projeto
**Nome do Projeto:** self-docs 
**Descrição:** Uma plataforma de gestão de conhecimento pessoal, desenvolvida para rodar localmente (self-hosted) via Docker. O objetivo é fornecer um hub seguro, organizado e rápido para documentações profissionais, anotações de projetos, guias, cheat sheets e notas privadas, tudo baseado na estrutura de arquivos Markdown.
**Referência de Design:** O layout base e a disposição dos elementos devem seguir fielmente o arquivo com todas as diretivas e referencias (`DESIGN.md`).

## 2. Arquitetura e Stack Tecnológico
A aplicação seguirá uma arquitetura cliente-servidor tradicional, totalmente conteinerizada.
*   **Frontend:** React configurado com Vite.
*   **Backend:** Golang utilizando o framework Gin.
*   **Banco de Dados:** PostgreSQL (para armazenar metadados, conteúdo dos documentos, tags e logs de atividades).
*   **Cache (Opcional/Recomendado):** Redis (para otimizar a busca e o carregamento da página inicial).
*   **Infraestrutura/Deploy:** Docker e Docker Compose (orquestrando os containers do Frontend, Backend, DB e Cache).

## 3. Funcionalidades Principais (Core Features)

### 3.1. Gestão de Documentos (Criação e Edição)
*   **Editor Embutido:** A plataforma deve possuir um editor web integrado (WYSIWYG ou editor de Markdown em tempo real) para criação e edição dos arquivos diretamente pela interface.
*   **Importação de Markdown:** O sistema deve oferecer a opção de importar arquivos `.md` pré-existentes da máquina local do usuário para dentro da plataforma.
*   **Tags:** Todo documento deve suportar a adição de tags para facilitar a categorização e a busca.

### 3.2. Seções de Conteúdo
A plataforma será dividida em 4 pilares principais:
1.  **Workflows & Guides:** Documentos únicos contendo procedimentos padrões e guias passo a passo.
2.  **Project Notes:** Estrutura hierárquica (semelhante ao Docusaurus), permitindo documentações amplas com sub-itens e aninhamento de páginas.
3.  **Cheat Sheets:** Listas de referências rápidas, comandos e snippets de código.
4.  **Private Archive (Notas Privadas):** Documentos confidenciais e registros pessoais. 
    *   **Segurança:** Acesso protegido. Ao clicar no card ou tentar acessar a URL desta seção, um **modal/popup** deve solicitar uma **Senha Mestre**.
    *   **Isolamento:** O conteúdo desta seção NÃO deve aparecer em listagens públicas da home ou na busca global.

### 3.3. Busca Global (Full-Text Search)
*   Um campo de "input" global no topo da página.
*   A busca deve realizar um *full-text search*, pesquisando por termos não apenas nos títulos e tags, mas também no **conteúdo interno** dos documentos.
*   **Restrição Crítica:** A busca deve ignorar totalmente os documentos pertencentes à seção "Private Archive".

### 3.4. Dashboard (Página Inicial)
A página inicial deve centralizar o acesso às áreas principais e exibir resumos dinâmicos:
*   **Cards de Navegação:** 4 cards principais levando para as seções descritas no item 3.2.
*   **Recently Updated (Atualizados Recentemente):** Uma lista com os últimos arquivos modificados (exibindo título, tempo da última alteração e link). *Deve excluir arquivos do Private Archive.*
*   **Popular Tags (Tags Populares):** Um agrupamento em formato de "pills" ou "badges" das tags mais utilizadas na plataforma para filtragem rápida.
*   **Activity Feed (Feed de Atividades):** Um log cronológico simples de ações do usuário (ex: "Criou Cheat Sheet: Git Commands", "Atualizou o Guia de Frontend").

## 4. Requisitos de UI/UX (Baseado na imagem de referência)
*   **Cabeçalho (Header):**
    *   Logotipo "SelfDocs" à esquerda.
    *   Navegação principal centralizada/alinhada à direita (Home, Documents, Knowledge Base, Private Notes, Settings).
    *   *Nota:* O ícone de notificações (sino) presente no design original foi **descartado do escopo**.
    *   Avatar do usuário com menu dropdown (focado apenas em configurações globais, sem necessidade de gestão complexa de perfil).
*   **Área de Boas-Vindas:** Saudação ("Welcome back") e input de busca global.
*   **Rodapé (Footer):** Mapa do site simplificado, links para portal de Admin (se houver no futuro) e API, informações de copyright.

## 5. Requisitos Não Funcionais
*   **Local First:** O sistema não requer sistema de login convencional com múltiplos usuários; assume-se que é de uso único, rodando na máquina ou servidor pessoal do usuário.
*   **Performance:** A busca global e o carregamento do "Recently Updated" devem ser altamente responsivos (uso do Redis é incentivado aqui).
*   **Segurança (Cofre Privado):** A senha mestre deve ser validada de forma segura (hash armazenado no banco). O conteúdo privado, no futuro, pode ser criptografado em repouso no banco de dados para segurança adicional.

## 6. Próximos Passos Sugeridos para a LLM Desenvolvedora
1.  **Modelagem do Banco de Dados:** Criar os schemas SQL para `documents`, `tags`, `document_tags` e `activity_logs`.
2.  **API REST (Go/Gin):** Desenvolver os endpoints CRUD para os documentos, endpoints de busca com suporte a *full-text search* no PostgreSQL (ex: `to_tsvector`), e endpoint de validação da senha mestre.
3.  **Frontend (React/Vite):** Estruturar o roteamento, criar os componentes de UI base (Cards, Listas, Modal de Senha, Input de Busca) e integrar um editor Markdown (como o MDEditor ou Milkdown).
4.  **Dockerização:** Configurar os `Dockerfile`s para o React e o Go, e estruturar o `docker-compose.yml` para subir toda a stack unificada.
