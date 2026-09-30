-- Seed Data for Testing Bikko Platform (PostgreSQL 16 + PostGIS)
-- Populated directly from Android Mock JSON files (categories.json, services.json, bikkers.json, client.json, solicitations.json)

-- 1. Insert Categories
INSERT INTO categories (id, name, icon_url, description, is_active) VALUES
('3f8a9b21-0000-0000-0000-000000000001', 'Pedreiro / Reformas', 'https://images.pexels.com/photos/1643383/pexels-photo-1643383.jpeg', 'Serviços de alvenaria, construção e reformas residenciais', true),
('a7c2d4e1-0000-0000-0000-000000000002', 'Pintor', 'https://images.pexels.com/photos/4505938/pexels-photo-4505938.jpeg', 'Pintura residencial, comercial e acabamento fino', true),
('b91e2c44-0000-0000-0000-000000000003', 'Eletricista', 'https://images.pexels.com/photos/5261835/pexels-photo-5261835.jpeg', 'Instalações elétricas, quadros de luz e manutenção 24h', true),
('c84f12aa-0000-0000-0000-000000000004', 'Encanador / Hidráulica', 'https://images.pexels.com/photos/4792249/pexels-photo-4792249.jpeg', 'Caça vazamentos, desentupimentos e hidráulica geral', true),
('d72ab993-0000-0000-0000-000000000005', 'Mudanças e Carretos', 'https://images.pexels.com/photos/58704/moving-boxes-cardboard-boxes-house-58704.jpeg', 'Mudanças residenciais, comerciais e transporte seguro', true),
('e56bc118-0000-0000-0000-000000000006', 'Montador de Móveis', 'https://images.pexels.com/photos/3825586/pexels-photo-3825586.jpeg', 'Montagem e desmontagem de móveis planejados e convencionais', true),
('f01dd882-0000-0000-0000-000000000007', 'Marido de Aluguel', 'https://images.pexels.com/photos/4969409/pexels-photo-4969409.jpeg', 'Pequenos reparos, instalações diversas e manutenção doméstica', true),
('1aa93cfe-0000-0000-0000-000000000008', 'Diarista / Faxina', 'https://images.pexels.com/photos/3747031/pexels-photo-3747031.jpeg', 'Faxina residencial completa, limpeza comercial e pós-obra', true),
('29bc44de-0000-0000-0000-000000000009', 'Técnico de Informática', 'https://images.pexels.com/photos/1181406/pexels-photo-1181406.jpeg', 'Manutenção de computadores, redes Wi-Fi e suporte TI', true),
('3cc18f77-0000-0000-0000-000000000010', 'Frete / Transporte Local', 'https://images.pexels.com/photos/949670/pexels-photo-949670.jpeg', 'Transporte rápido de objetos e cargas urbanas', true),
('4de2b7ac-0000-0000-0000-000000000011', 'Aulas de Violão', 'https://images.pexels.com/photos/374870/pexels-photo-374870.jpeg', 'Aulas particulares de instrumentos musicais para iniciantes', true),
('00000000-0000-0000-0000-000000000099', 'Geral', 'https://images.pexels.com/photos/1643383/pexels-photo-1643383.jpeg', 'Outros serviços e utilidades gerais', true)
ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  icon_url = EXCLUDED.icon_url,
  description = EXCLUDED.description;

-- 2. Insert Users (Customers, Bikkers & Mock Reviewers)
INSERT INTO users (id, full_name, email, phone, cpf, password_hash, is_bikker, rating, total_ratings, profile_photo_url) VALUES
-- Main Customer
('11111111-0000-0000-0000-000000000099', 'João da Silva', 'joao.silva@exemplo.com', '(11) 98765-4321', '111.222.333-44', '$2a$10$vN4kY8v6YxW0Nn2/R2R3eeS.p01G5XmZ41sM2Y6Z41sM2Y6Z41sM2', false, 4.80, 12, 'https://images.pexels.com/photos/774909/pexels-photo-774909.jpeg'),
-- Bikkers
('11111111-0000-0000-0000-000000000001', 'Ricardo Almeida', 'ricardo.almeida@bikko.com.br', '(11) 99999-1001', '222.333.444-01', '$2a$10$vN4kY8v6YxW0Nn2/R2R3eeS.p01G5XmZ41sM2Y6Z41sM2Y6Z41sM2', true, 4.80, 105, 'https://images.pexels.com/photos/220453/pexels-photo-220453.jpeg'),
('11111111-0000-0000-0000-000000000002', 'Juliana Souza', 'juliana.souza@bikko.com.br', '(11) 99999-1002', '222.333.444-02', '$2a$10$vN4kY8v6YxW0Nn2/R2R3eeS.p01G5XmZ41sM2Y6Z41sM2Y6Z41sM2', true, 4.90, 85, 'https://images.pexels.com/photos/774909/pexels-photo-774909.jpeg'),
('11111111-0000-0000-0000-000000000003', 'Pedro Pedreiro', 'pedro.pedreiro@bikko.com.br', '(11) 99999-1003', '222.333.444-03', '$2a$10$vN4kY8v6YxW0Nn2/R2R3eeS.p01G5XmZ41sM2Y6Z41sM2Y6Z41sM2', true, 4.90, 40, 'https://images.pexels.com/photos/91227/pexels-photo-91227.jpeg'),
('11111111-0000-0000-0000-000000000004', 'Marcos Pintor', 'marcos.pintor@bikko.com.br', '(11) 99999-1004', '222.333.444-04', '$2a$10$vN4kY8v6YxW0Nn2/R2R3eeS.p01G5XmZ41sM2Y6Z41sM2Y6Z41sM2', true, 4.80, 18, 'https://images.pexels.com/photos/614810/pexels-photo-614810.jpeg'),
('11111111-0000-0000-0000-000000000005', 'Roberto Carreto', 'roberto.carreto@bikko.com.br', '(11) 99999-1005', '222.333.444-05', '$2a$10$vN4kY8v6YxW0Nn2/R2R3eeS.p01G5XmZ41sM2Y6Z41sM2Y6Z41sM2', true, 4.60, 15, 'https://images.pexels.com/photos/220453/pexels-photo-220453.jpeg'),
('11111111-0000-0000-0000-000000000006', 'Tiago Montador', 'tiago.montador@bikko.com.br', '(11) 99999-1006', '222.333.444-06', '$2a$10$vN4kY8v6YxW0Nn2/R2R3eeS.p01G5XmZ41sM2Y6Z41sM2Y6Z41sM2', true, 4.90, 22, 'https://images.pexels.com/photos/220453/pexels-photo-220453.jpeg'),
('11111111-0000-0000-0000-000000000007', 'Lucas Silva', 'lucas.ti@bikko.com.br', '(11) 99999-1007', '222.333.444-07', '$2a$10$vN4kY8v6YxW0Nn2/R2R3eeS.p01G5XmZ41sM2Y6Z41sM2Y6Z41sM2', true, 4.80, 12, 'https://images.pexels.com/photos/614810/pexels-photo-614810.jpeg'),
('11111111-0000-0000-0000-000000000008', 'André Santos', 'andre.musica@bikko.com.br', '(11) 99999-1008', '222.333.444-08', '$2a$10$vN4kY8v6YxW0Nn2/R2R3eeS.p01G5XmZ41sM2Y6Z41sM2Y6Z41sM2', true, 5.00, 9, 'https://images.pexels.com/photos/91227/pexels-photo-91227.jpeg'),
('11111111-0000-0000-0000-000000000009', 'Carlos Mendes', 'carlos.mendes@bikko.com.br', '(11) 99999-1009', '222.333.444-09', '$2a$10$vN4kY8v6YxW0Nn2/R2R3eeS.p01G5XmZ41sM2Y6Z41sM2Y6Z41sM2', true, 4.90, 30, 'https://images.pexels.com/photos/220453/pexels-photo-220453.jpeg'),
-- Reviewer Clients from services.json
('11111111-0000-0000-0000-000000000021', 'Fernanda Lima', 'fernanda.lima@exemplo.com', '(11) 98111-0021', '333.444.555-21', '$2a$10$vN4kY8v6YxW0Nn2/R2R3eeS.p01G5XmZ41sM2Y6Z41sM2Y6Z41sM2', false, 5.00, 1, 'https://images.pexels.com/photos/415829/pexels-photo-415829.jpeg'),
('11111111-0000-0000-0000-000000000022', 'Rafael Almeida', 'rafael.almeida@exemplo.com', '(11) 98111-0022', '333.444.555-22', '$2a$10$vN4kY8v6YxW0Nn2/R2R3eeS.p01G5XmZ41sM2Y6Z41sM2Y6Z41sM2', false, 5.00, 2, 'https://images.pexels.com/photos/614810/pexels-photo-614810.jpeg'),
('11111111-0000-0000-0000-000000000023', 'Patrícia Martins', 'patricia.martins@exemplo.com', '(11) 98111-0023', '333.444.555-23', '$2a$10$vN4kY8v6YxW0Nn2/R2R3eeS.p01G5XmZ41sM2Y6Z41sM2Y6Z41sM2', false, 5.00, 1, 'https://images.pexels.com/photos/1239291/pexels-photo-1239291.jpeg'),
('11111111-0000-0000-0000-000000000024', 'Lucas Ferreira', 'lucas.ferreira@exemplo.com', '(11) 98111-0024', '333.444.555-24', '$2a$10$vN4kY8v6YxW0Nn2/R2R3eeS.p01G5XmZ41sM2Y6Z41sM2Y6Z41sM2', false, 5.00, 1, 'https://images.pexels.com/photos/1043474/pexels-photo-1043474.jpeg'),
('11111111-0000-0000-0000-000000000025', 'Camila Rodrigues', 'camila.rodrigues@exemplo.com', '(11) 98111-0025', '333.444.555-25', '$2a$10$vN4kY8v6YxW0Nn2/R2R3eeS.p01G5XmZ41sM2Y6Z41sM2Y6Z41sM2', false, 5.00, 2, 'https://images.pexels.com/photos/733872/pexels-photo-733872.jpeg'),
('11111111-0000-0000-0000-000000000026', 'Maria Silva', 'maria.silva@exemplo.com', '(11) 98111-0026', '333.444.555-26', '$2a$10$vN4kY8v6YxW0Nn2/R2R3eeS.p01G5XmZ41sM2Y6Z41sM2Y6Z41sM2', false, 5.00, 3, 'https://images.pexels.com/photos/774909/pexels-photo-774909.jpeg'),
('11111111-0000-0000-0000-000000000027', 'Carlos Henrique', 'carlos.henrique@exemplo.com', '(11) 98111-0027', '333.444.555-27', '$2a$10$vN4kY8v6YxW0Nn2/R2R3eeS.p01G5XmZ41sM2Y6Z41sM2Y6Z41sM2', false, 5.00, 3, 'https://images.pexels.com/photos/220453/pexels-photo-220453.jpeg')
ON CONFLICT (id) DO UPDATE SET
  full_name = EXCLUDED.full_name,
  email = EXCLUDED.email,
  profile_photo_url = EXCLUDED.profile_photo_url;

-- 3. Insert Bikkers (Geospatial Points in São Paulo)
INSERT INTO bikkers (id, profession, experience_years, description, location_name, location_geom, rating, total_reviews, is_active) VALUES
('11111111-0000-0000-0000-000000000001', 'Encanador e Eletricista Senior', '10 anos', 'Especialista em manutenção residencial com foco em rapidez e limpeza. Atendimento 24h para emergências elétricas e hidráulicas.', 'São Paulo, SP', ST_SetSRID(ST_MakePoint(-46.63330, -23.55052), 4326)::geography, 4.80, 105, true),
('11111111-0000-0000-0000-000000000002', 'Diarista Profissional', '5 anos', 'Limpeza residencial e comercial detalhada com capricho, organização e agilidade. Referências excelentes.', 'São Paulo, SP', ST_SetSRID(ST_MakePoint(-46.65588, -23.56141), 4326)::geography, 4.90, 85, true),
('11111111-0000-0000-0000-000000000003', 'Pedreiro e Construtor', '12 anos', 'Construções, fundações, acabamentos, reboco e assentamento de porcelanato com alto padrão.', 'São Paulo, SP', ST_SetSRID(ST_MakePoint(-46.68900, -23.56100), 4326)::geography, 4.90, 40, true),
('11111111-0000-0000-0000-000000000004', 'Pintor e Decorador', '6 anos', 'Serviços de pintura interna e externa, aplicação de massas e texturas com acabamento fino.', 'São Paulo, SP', ST_SetSRID(ST_MakePoint(-46.64000, -23.54000), 4326)::geography, 4.80, 18, true),
('11111111-0000-0000-0000-000000000005', 'Carretos e Transportes', '7 anos', 'Transporte seguro e ágil de mudanças residenciais e comerciais.', 'São Paulo, SP', ST_SetSRID(ST_MakePoint(-46.62000, -23.53000), 4326)::geography, 4.60, 15, true),
('11111111-0000-0000-0000-000000000006', 'Montador de Móveis', '4 anos', 'Montagem de guarda-roupas, armários de cozinha, camas e móveis comprados na internet.', 'São Paulo, SP', ST_SetSRID(ST_MakePoint(-46.65000, -23.57000), 4326)::geography, 4.90, 22, true),
('11111111-0000-0000-0000-000000000007', 'Técnico de Informática', '5 anos', 'Configuração de redes Wi-Fi, remoção de vírus, formatação e manutenção de computadores.', 'São Paulo, SP', ST_SetSRID(ST_MakePoint(-46.66000, -23.58000), 4326)::geography, 4.80, 12, true),
('11111111-0000-0000-0000-000000000008', 'Professor de Violão', '8 anos', 'Aulas presenciais para iniciantes. Aprenda acordes básicos, ritmos e suas primeiras músicas.', 'São Paulo, SP', ST_SetSRID(ST_MakePoint(-46.67000, -23.59000), 4326)::geography, 5.00, 9, true),
('11111111-0000-0000-0000-000000000009', 'Técnico de Climatização', '6 anos', 'Instalação e manutenção preventiva de ar condicionado residencial e comercial.', 'São Paulo, SP', ST_SetSRID(ST_MakePoint(-46.63500, -23.55500), 4326)::geography, 4.90, 30, true)
ON CONFLICT (id) DO UPDATE SET
  profession = EXCLUDED.profession,
  description = EXCLUDED.description,
  rating = EXCLUDED.rating;

-- 4. Insert Services
INSERT INTO services (id, bikker_id, category_id, name, description, thumbnail_url, location_geom, reviews_average, total_reviews, is_active) VALUES
('22222222-0000-0000-0000-000000000001', '11111111-0000-0000-0000-000000000003', '3f8a9b21-0000-0000-0000-000000000001', 'Construção e Alvenaria', 'Construção de muros, reboco, assentamento de pisos e azulejos, reformas residenciais em geral com fino acabamento.', 'https://images.pexels.com/photos/2219024/pexels-photo-2219024.jpeg', ST_SetSRID(ST_MakePoint(-46.68900, -23.56100), 4326)::geography, 4.90, 2, true),
('22222222-0000-0000-0000-000000000002', '11111111-0000-0000-0000-000000000004', 'a7c2d4e1-0000-0000-0000-000000000002', 'Pintura Residencial e Comercial', 'Pintura de paredes internas, externas, tetos, aplicação de massa corrida, textura e verniz. Serviço limpo e rápido.', 'https://images.pexels.com/photos/6474136/pexels-photo-6474136.jpeg', ST_SetSRID(ST_MakePoint(-46.64000, -23.54000), 4326)::geography, 4.80, 1, true),
('22222222-0000-0000-0000-000000000003', '11111111-0000-0000-0000-000000000001', 'b91e2c44-0000-0000-0000-000000000003', 'Instalação e Manutenção Elétrica', 'Instalação de chuveiros, luminárias, ventiladores de teto, tomadas, disjuntores e manutenção em fiação elétrica.', 'https://images.pexels.com/photos/1435183/pexels-photo-1435183.jpeg', ST_SetSRID(ST_MakePoint(-46.63330, -23.55052), 4326)::geography, 4.90, 2, true),
('22222222-0000-0000-0000-000000000004', '11111111-0000-0000-0000-000000000001', 'c84f12aa-0000-0000-0000-000000000004', 'Encanador 24h & Caça Vazamentos', 'Reparo de vazamentos em canos, torneiras, registros. Desentupimento de pias e vasos sanitários. Instalação de metais e louças.', 'https://images.pexels.com/photos/2310904/pexels-photo-2310904.jpeg', ST_SetSRID(ST_MakePoint(-46.63330, -23.55052), 4326)::geography, 4.70, 1, true),
('22222222-0000-0000-0000-000000000005', '11111111-0000-0000-0000-000000000005', 'd72ab993-0000-0000-0000-000000000005', 'Carretos e Mudanças com Montagem', 'Realizamos mudanças residenciais e comerciais. Temos ajudante e serviço completo de embalagem e transporte seguro.', 'https://images.pexels.com/photos/7464687/pexels-photo-7464687.jpeg', ST_SetSRID(ST_MakePoint(-46.62000, -23.53000), 4326)::geography, 4.60, 1, true),
('22222222-0000-0000-0000-000000000006', '11111111-0000-0000-0000-000000000006', 'e56bc118-0000-0000-0000-000000000006', 'Montador de Móveis Profissional', 'Montagem de guarda-roupas, armários de cozinha, camas, estantes, painéis de TV e móveis comprados na internet.', 'https://images.pexels.com/photos/3825586/pexels-photo-3825586.jpeg', ST_SetSRID(ST_MakePoint(-46.65000, -23.57000), 4326)::geography, 4.90, 1, true),
('22222222-0000-0000-0000-000000000007', '11111111-0000-0000-0000-000000000001', 'f01dd882-0000-0000-0000-000000000007', 'Pequenos Reparos Residenciais', 'Serviços gerais: fixação de quadros, prateleiras, cortinas, troca de lâmpadas, regulagem de portas de armários e reparos diversos.', 'https://images.pexels.com/photos/4969409/pexels-photo-4969409.jpeg', ST_SetSRID(ST_MakePoint(-46.63330, -23.55052), 4326)::geography, 4.80, 1, true),
('22222222-0000-0000-0000-000000000008', '11111111-0000-0000-0000-000000000002', '1aa93cfe-0000-0000-0000-000000000008', 'Faxina Residencial Completa', 'Limpeza detalhada de salas, quartos, banheiros e cozinha. Lavagem de louças e organização do lar com capricho e confiança.', 'https://images.pexels.com/photos/4099471/pexels-photo-4099471.jpeg', ST_SetSRID(ST_MakePoint(-46.65588, -23.56141), 4326)::geography, 4.90, 2, true),
('22222222-0000-0000-0000-000000000009', '11111111-0000-0000-0000-000000000007', '29bc44de-0000-0000-0000-000000000009', 'Formatação, Redes e Suporte TI', 'Configuração de roteadores Wi-Fi, remoção de vírus, formatação com backup, limpeza interna de computadores e notebooks.', 'https://images.pexels.com/photos/356056/pexels-photo-356056.jpeg', ST_SetSRID(ST_MakePoint(-46.66000, -23.58000), 4326)::geography, 4.80, 1, true),
('22222222-0000-0000-0000-000000000010', '11111111-0000-0000-0000-000000000005', '3cc18f77-0000-0000-0000-000000000010', 'Fretes em Geral (Pick-up)', 'Transporte de pequenas cargas, sofás, eletrodomésticos, material de construção. Serviço ágil e pontual na região urbana.', 'https://images.pexels.com/photos/949670/pexels-photo-949670.jpeg', ST_SetSRID(ST_MakePoint(-46.62000, -23.53000), 4326)::geography, 4.50, 0, true),
('22222222-0000-0000-0000-000000000011', '11111111-0000-0000-0000-000000000008', '4de2b7ac-0000-0000-0000-000000000011', 'Aulas Práticas de Violão Iniciante', 'Aulas presenciais para iniciantes. Aprenda acordes básicos, ritmos e suas primeiras músicas de forma simples e divertida.', 'https://images.pexels.com/photos/374870/pexels-photo-374870.jpeg', ST_SetSRID(ST_MakePoint(-46.67000, -23.59000), 4326)::geography, 5.00, 1, true),
('22222222-0000-0000-0000-000000000012', '11111111-0000-0000-0000-000000000009', 'f01dd882-0000-0000-0000-000000000007', 'Instalação de Ar Condicionado', 'Instalação de split de 12000 BTUs na sala de estar. Necessário furar a parede e passar tubulação.', 'https://images.pexels.com/photos/2310904/pexels-photo-2310904.jpeg', ST_SetSRID(ST_MakePoint(-46.63500, -23.55500), 4326)::geography, 4.90, 1, true)
ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  description = EXCLUDED.description,
  reviews_average = EXCLUDED.reviews_average;

-- Clear transient operational seed tables to ensure complete idempotency
DELETE FROM reviews;
DELETE FROM quote_negotiations;
DELETE FROM orders;
DELETE FROM quote_requests;
DELETE FROM service_photos;

-- 5. Insert Service Photos (Directly from services.json)
INSERT INTO service_photos (id, service_id, photo_url, display_order) VALUES
-- Pedreiro
(uuid_generate_v4(), '22222222-0000-0000-0000-000000000001', 'https://images.pexels.com/photos/2219024/pexels-photo-2219024.jpeg', 1),
(uuid_generate_v4(), '22222222-0000-0000-0000-000000000001', 'https://images.pexels.com/photos/585418/pexels-photo-585418.jpeg', 2),
-- Pintor
(uuid_generate_v4(), '22222222-0000-0000-0000-000000000002', 'https://images.pexels.com/photos/6474136/pexels-photo-6474136.jpeg', 1),
(uuid_generate_v4(), '22222222-0000-0000-0000-000000000002', 'https://images.pexels.com/photos/6474127/pexels-photo-6474127.jpeg', 2),
-- Eletricista
(uuid_generate_v4(), '22222222-0000-0000-0000-000000000003', 'https://images.pexels.com/photos/1435183/pexels-photo-1435183.jpeg', 1),
(uuid_generate_v4(), '22222222-0000-0000-0000-000000000003', 'https://images.pexels.com/photos/5691612/pexels-photo-5691612.jpeg', 2),
-- Encanador
(uuid_generate_v4(), '22222222-0000-0000-0000-000000000004', 'https://images.pexels.com/photos/2310904/pexels-photo-2310904.jpeg', 1),
(uuid_generate_v4(), '22222222-0000-0000-0000-000000000004', 'https://images.pexels.com/photos/4792249/pexels-photo-4792249.jpeg', 2),
-- Mudanças e Carretos
(uuid_generate_v4(), '22222222-0000-0000-0000-000000000005', 'https://images.pexels.com/photos/7464687/pexels-photo-7464687.jpeg', 1),
(uuid_generate_v4(), '22222222-0000-0000-0000-000000000005', 'https://images.pexels.com/photos/7464673/pexels-photo-7464673.jpeg', 2),
-- Montador de Móveis
(uuid_generate_v4(), '22222222-0000-0000-0000-000000000006', 'https://images.pexels.com/photos/3825586/pexels-photo-3825586.jpeg', 1),
(uuid_generate_v4(), '22222222-0000-0000-0000-000000000006', 'https://images.pexels.com/photos/3825582/pexels-photo-3825582.jpeg', 2),
-- Marido de Aluguel
(uuid_generate_v4(), '22222222-0000-0000-0000-000000000007', 'https://images.pexels.com/photos/4969409/pexels-photo-4969409.jpeg', 1),
-- Diarista / Faxina
(uuid_generate_v4(), '22222222-0000-0000-0000-000000000008', 'https://images.pexels.com/photos/4099471/pexels-photo-4099471.jpeg', 1),
(uuid_generate_v4(), '22222222-0000-0000-0000-000000000008', 'https://images.pexels.com/photos/3747031/pexels-photo-3747031.jpeg', 2),
-- Técnico de TI
(uuid_generate_v4(), '22222222-0000-0000-0000-000000000009', 'https://images.pexels.com/photos/356056/pexels-photo-356056.jpeg', 1),
(uuid_generate_v4(), '22222222-0000-0000-0000-000000000009', 'https://images.pexels.com/photos/1181406/pexels-photo-1181406.jpeg', 2),
-- Frete
(uuid_generate_v4(), '22222222-0000-0000-0000-000000000010', 'https://images.pexels.com/photos/949670/pexels-photo-949670.jpeg', 1),
-- Aulas de Violão
(uuid_generate_v4(), '22222222-0000-0000-0000-000000000011', 'https://images.pexels.com/photos/374870/pexels-photo-374870.jpeg', 1),
(uuid_generate_v4(), '22222222-0000-0000-0000-000000000011', 'https://images.pexels.com/photos/1407322/pexels-photo-1407322.jpeg', 2),
-- Instalação de Ar Condicionado
(uuid_generate_v4(), '22222222-0000-0000-0000-000000000012', 'https://images.pexels.com/photos/2310904/pexels-photo-2310904.jpeg', 1);

-- 6. Insert Quote Requests (4 Solicitations + 13 for Service Reviews)
INSERT INTO quote_requests (id, customer_id, bikker_id, service_id, status, initial_description, desired_date, location_address, location_geom) VALUES
-- sol_1 (Instalação de Ar Condicionado - Carlos Mendes)
('33333333-0000-0000-0000-000000000001', '11111111-0000-0000-0000-000000000099', '11111111-0000-0000-0000-000000000009', '22222222-0000-0000-0000-000000000012', 'NEGOTIATING', 'Instalação de split de 12000 BTUs na sala de estar. Necessário furar a parede e passar tubulação.', '2024-06-24 14:00:00+00', 'Rua da Consolação, 1500', ST_SetSRID(ST_MakePoint(-46.63500, -23.55500), 4326)::geography),
-- sol_2 (Pintura de Quarto - Marcos Pintor)
('33333333-0000-0000-0000-000000000002', '11111111-0000-0000-0000-000000000099', '11111111-0000-0000-0000-000000000004', '22222222-0000-0000-0000-000000000002', 'ACCEPTED', 'Pintura de duas paredes da sala com tinta acrílica fosca. Tinta e materiais já comprados.', '2024-06-26 08:00:00+00', 'Alameda Santos, 800', ST_SetSRID(ST_MakePoint(-46.64000, -23.54000), 4326)::geography),
-- sol_3 (Reparo de Disjuntor - Ricardo Almeida)
('33333333-0000-0000-0000-000000000003', '11111111-0000-0000-0000-000000000099', '11111111-0000-0000-0000-000000000001', '22222222-0000-0000-0000-000000000003', 'REJECTED', 'Disjuntor geral caindo ao ligar o chuveiro elétrico. Necessário avaliar fiação e disjuntor.', '2024-06-18 10:00:00+00', 'Av. Brigadeiro Luís Antônio, 2000', ST_SetSRID(ST_MakePoint(-46.63330, -23.55052), 4326)::geography),
-- sol_4 (Limpeza Residencial - Juliana Souza)
('33333333-0000-0000-0000-000000000004', '11111111-0000-0000-0000-000000000099', '11111111-0000-0000-0000-000000000002', '22222222-0000-0000-0000-000000000008', 'ACCEPTED', 'Limpeza completa de apartamento de 2 quartos, sala, cozinha e 2 banheiros.', '2024-06-15 09:00:00+00', 'Rua Bela Cintra, 950', ST_SetSRID(ST_MakePoint(-46.65588, -23.56141), 4326)::geography),

-- Quote Requests corresponding to completed jobs for reviews in services.json
('33333333-0000-0000-0000-000000000011', '11111111-0000-0000-0000-000000000021', '11111111-0000-0000-0000-000000000003', '22222222-0000-0000-0000-000000000001', 'ACCEPTED', 'Reforma de banheiro residencial com troca de pisos', '2024-06-01 09:00:00+00', 'Rua Oscar Freire, 300', ST_SetSRID(ST_MakePoint(-46.68900, -23.56100), 4326)::geography),
('33333333-0000-0000-0000-000000000012', '11111111-0000-0000-0000-000000000022', '11111111-0000-0000-0000-000000000003', '22222222-0000-0000-0000-000000000001', 'ACCEPTED', 'Construção de muro e reboco externo', '2024-05-28 08:00:00+00', 'Av. Rebouças, 1200', ST_SetSRID(ST_MakePoint(-46.68900, -23.56100), 4326)::geography),
('33333333-0000-0000-0000-000000000013', '11111111-0000-0000-0000-000000000023', '11111111-0000-0000-0000-000000000004', '22222222-0000-0000-0000-000000000002', 'ACCEPTED', 'Pintura completa de sala e teto', '2024-05-20 08:30:00+00', 'Rua Pamplona, 450', ST_SetSRID(ST_MakePoint(-46.64000, -23.54000), 4326)::geography),
('33333333-0000-0000-0000-000000000014', '11111111-0000-0000-0000-000000000024', '11111111-0000-0000-0000-000000000001', '22222222-0000-0000-0000-000000000003', 'ACCEPTED', 'Emergência de curto-circuito na cozinha', '2024-06-05 15:00:00+00', 'Rua Haddock Lobo, 800', ST_SetSRID(ST_MakePoint(-46.63330, -23.55052), 4326)::geography),
('33333333-0000-0000-0000-000000000015', '11111111-0000-0000-0000-000000000025', '11111111-0000-0000-0000-000000000001', '22222222-0000-0000-0000-000000000003', 'ACCEPTED', 'Instalação de iluminação LED e spots de teto', '2024-06-02 10:00:00+00', 'Alameda Lorena, 1100', ST_SetSRID(ST_MakePoint(-46.63330, -23.55052), 4326)::geography),
('33333333-0000-0000-0000-000000000016', '11111111-0000-0000-0000-000000000026', '11111111-0000-0000-0000-000000000001', '22222222-0000-0000-0000-000000000004', 'ACCEPTED', 'Reparo em vazamento de cano e torneira do banheiro', '2024-06-04 11:00:00+00', 'Av. Paulista, 1000', ST_SetSRID(ST_MakePoint(-46.63330, -23.55052), 4326)::geography),
('33333333-0000-0000-0000-000000000017', '11111111-0000-0000-0000-000000000027', '11111111-0000-0000-0000-000000000005', '22222222-0000-0000-0000-000000000005', 'ACCEPTED', 'Mudança residencial e transporte de caixas', '2024-05-25 08:00:00+00', 'Rua Vergueiro, 1500', ST_SetSRID(ST_MakePoint(-46.62000, -23.53000), 4326)::geography),
('33333333-0000-0000-0000-000000000018', '11111111-0000-0000-0000-000000000027', '11111111-0000-0000-0000-000000000006', '22222222-0000-0000-0000-000000000006', 'ACCEPTED', 'Montagem de guarda-roupas planejado', '2024-06-03 14:00:00+00', 'Rua Domingos de Morais, 900', ST_SetSRID(ST_MakePoint(-46.65000, -23.57000), 4326)::geography),
('33333333-0000-0000-0000-000000000019', '11111111-0000-0000-0000-000000000026', '11111111-0000-0000-0000-000000000001', '22222222-0000-0000-0000-000000000007', 'ACCEPTED', 'Instalação de cortinas e suporte de TV na parede', '2024-05-26 16:00:00+00', 'Av. Angélica, 700', ST_SetSRID(ST_MakePoint(-46.63330, -23.55052), 4326)::geography),
('33333333-0000-0000-0000-000000000020', '11111111-0000-0000-0000-000000000027', '11111111-0000-0000-0000-000000000002', '22222222-0000-0000-0000-000000000008', 'ACCEPTED', 'Faxina semanal completa no apartamento', '2024-06-04 08:30:00+00', 'Rua Bela Cintra, 1200', ST_SetSRID(ST_MakePoint(-46.65588, -23.56141), 4326)::geography),
('33333333-0000-0000-0000-000000000021', '11111111-0000-0000-0000-000000000025', '11111111-0000-0000-0000-000000000002', '22222222-0000-0000-0000-000000000008', 'ACCEPTED', 'Faxina detalhada pós-obra de gesso e pintura', '2024-05-27 08:00:00+00', 'Rua Teodoro Sampaio, 850', ST_SetSRID(ST_MakePoint(-46.65588, -23.56141), 4326)::geography),
('33333333-0000-0000-0000-000000000022', '11111111-0000-0000-0000-000000000022', '11111111-0000-0000-0000-000000000007', '22222222-0000-0000-0000-000000000009', 'ACCEPTED', 'Limpeza física, troca de pasta térmica e otimização de notebook', '2024-06-02 13:00:00+00', 'Rua da Consolação, 2000', ST_SetSRID(ST_MakePoint(-46.66000, -23.58000), 4326)::geography),
('33333333-0000-0000-0000-000000000023', '11111111-0000-0000-0000-000000000026', '11111111-0000-0000-0000-000000000008', '22222222-0000-0000-0000-000000000011', 'ACCEPTED', 'Aulas particulares práticas de violão iniciante', '2024-05-15 14:00:00+00', 'Rua Cardeal Arcoverde, 600', ST_SetSRID(ST_MakePoint(-46.67000, -23.59000), 4326)::geography);

-- 7. Insert Quote Negotiations (Photo attachments for sol_1)
INSERT INTO quote_negotiations (id, quote_request_id, sender_id, message, proposed_price, photos, is_counter_offer) VALUES
('66666666-0000-0000-0000-000000000001', '33333333-0000-0000-0000-000000000001', '11111111-0000-0000-0000-000000000009', 'Orçamento enviado para instalação completa com suporte e tubulação inclusos.', 350.00, '["https://images.pexels.com/photos/2310904/pexels-photo-2310904.jpeg"]'::jsonb, false);

-- 8. Insert Orders (4 Solicitations + 13 for Service Reviews)
INSERT INTO orders (id, quote_request_id, customer_id, bikker_id, service_id, status, final_price, scheduled_date, cancellation_reason) VALUES
-- sol_1: SCHEDULED (Carlos Mendes - Ar Condicionado)
('44444444-0000-0000-0000-000000000001', '33333333-0000-0000-0000-000000000001', '11111111-0000-0000-0000-000000000099', '11111111-0000-0000-0000-000000000009', '22222222-0000-0000-0000-000000000012', 'SCHEDULED', 350.00, '2024-06-24 14:00:00+00', NULL),
-- sol_2: IN_PROGRESS (Marcos Pintor - Pintura de Quarto)
('44444444-0000-0000-0000-000000000002', '33333333-0000-0000-0000-000000000002', '11111111-0000-0000-0000-000000000099', '11111111-0000-0000-0000-000000000004', '22222222-0000-0000-0000-000000000002', 'IN_PROGRESS', 850.00, '2024-06-26 08:00:00+00', NULL),
-- sol_3: CANCELLED (Ricardo Almeida - Reparo de Disjuntor)
('44444444-0000-0000-0000-000000000003', '33333333-0000-0000-0000-000000000003', '11111111-0000-0000-0000-000000000099', '11111111-0000-0000-0000-000000000001', '22222222-0000-0000-0000-000000000003', 'CANCELLED', 150.00, '2024-06-18 10:00:00+00', 'O provedor não tinha disponibilidade para a data/hora solicitada.'),
-- sol_4: COMPLETED (Juliana Souza - Limpeza Residencial)
('44444444-0000-0000-0000-000000000004', '33333333-0000-0000-0000-000000000004', '11111111-0000-0000-0000-000000000099', '11111111-0000-0000-0000-000000000002', '22222222-0000-0000-0000-000000000008', 'COMPLETED', 180.00, '2024-06-15 09:00:00+00', NULL),

-- Orders corresponding to completed jobs for reviews in services.json
('44444444-0000-0000-0000-000000000011', '33333333-0000-0000-0000-000000000011', '11111111-0000-0000-0000-000000000021', '11111111-0000-0000-0000-000000000003', '22222222-0000-0000-0000-000000000001', 'COMPLETED', 1200.00, CURRENT_TIMESTAMP - INTERVAL '3 days', NULL),
('44444444-0000-0000-0000-000000000012', '33333333-0000-0000-0000-000000000012', '11111111-0000-0000-0000-000000000022', '11111111-0000-0000-0000-000000000003', '22222222-0000-0000-0000-000000000001', 'COMPLETED', 950.00, CURRENT_TIMESTAMP - INTERVAL '7 days', NULL),
('44444444-0000-0000-0000-000000000013', '33333333-0000-0000-0000-000000000013', '11111111-0000-0000-0000-000000000023', '11111111-0000-0000-0000-000000000004', '22222222-0000-0000-0000-000000000002', 'COMPLETED', 750.00, CURRENT_TIMESTAMP - INTERVAL '14 days', NULL),
('44444444-0000-0000-0000-000000000014', '33333333-0000-0000-0000-000000000014', '11111111-0000-0000-0000-000000000024', '11111111-0000-0000-0000-000000000001', '22222222-0000-0000-0000-000000000003', 'COMPLETED', 220.00, CURRENT_TIMESTAMP - INTERVAL '1 day', NULL),
('44444444-0000-0000-0000-000000000015', '33333333-0000-0000-0000-000000000015', '11111111-0000-0000-0000-000000000025', '11111111-0000-0000-0000-000000000001', '22222222-0000-0000-0000-000000000003', 'COMPLETED', 350.00, CURRENT_TIMESTAMP - INTERVAL '5 days', NULL),
('44444444-0000-0000-0000-000000000016', '33333333-0000-0000-0000-000000000016', '11111111-0000-0000-0000-000000000026', '11111111-0000-0000-0000-000000000001', '22222222-0000-0000-0000-000000000004', 'COMPLETED', 190.00, CURRENT_TIMESTAMP - INTERVAL '2 days', NULL),
('44444444-0000-0000-0000-000000000017', '33333333-0000-0000-0000-000000000017', '11111111-0000-0000-0000-000000000027', '11111111-0000-0000-0000-000000000005', '22222222-0000-0000-0000-000000000005', 'COMPLETED', 600.00, CURRENT_TIMESTAMP - INTERVAL '7 days', NULL),
('44444444-0000-0000-0000-000000000018', '33333333-0000-0000-0000-000000000018', '11111111-0000-0000-0000-000000000027', '11111111-0000-0000-0000-000000000006', '22222222-0000-0000-0000-000000000006', 'COMPLETED', 280.00, CURRENT_TIMESTAMP - INTERVAL '3 days', NULL),
('44444444-0000-0000-0000-000000000019', '33333333-0000-0000-0000-000000000019', '11111111-0000-0000-0000-000000000026', '11111111-0000-0000-0000-000000000001', '22222222-0000-0000-0000-000000000007', 'COMPLETED', 150.00, CURRENT_TIMESTAMP - INTERVAL '7 days', NULL),
('44444444-0000-0000-0000-000000000020', '33333333-0000-0000-0000-000000000020', '11111111-0000-0000-0000-000000000027', '11111111-0000-0000-0000-000000000002', '22222222-0000-0000-0000-000000000008', 'COMPLETED', 180.00, CURRENT_TIMESTAMP - INTERVAL '2 days', NULL),
('44444444-0000-0000-0000-000000000021', '33333333-0000-0000-0000-000000000021', '11111111-0000-0000-0000-000000000025', '11111111-0000-0000-0000-000000000002', '22222222-0000-0000-0000-000000000008', 'COMPLETED', 250.00, CURRENT_TIMESTAMP - INTERVAL '7 days', NULL),
('44444444-0000-0000-0000-000000000022', '33333333-0000-0000-0000-000000000022', '11111111-0000-0000-0000-000000000022', '11111111-0000-0000-0000-000000000007', '22222222-0000-0000-0000-000000000009', 'COMPLETED', 160.00, CURRENT_TIMESTAMP - INTERVAL '4 days', NULL),
('44444444-0000-0000-0000-000000000023', '33333333-0000-0000-0000-000000000023', '11111111-0000-0000-0000-000000000026', '11111111-0000-0000-0000-000000000008', '22222222-0000-0000-0000-000000000011', 'COMPLETED', 100.00, CURRENT_TIMESTAMP - INTERVAL '21 days', NULL);

-- 9. Insert Reviews (Directly replicated from services.json)
INSERT INTO reviews (id, order_id, service_id, reviewer_id, bikker_id, rating, comment, like_count, dislike_count) VALUES
-- Pedreiro (2 reviews)
('55555555-0000-0000-0000-000000000001', '44444444-0000-0000-0000-000000000011', '22222222-0000-0000-0000-000000000001', '11111111-0000-0000-0000-000000000021', '11111111-0000-0000-0000-000000000003', 5, 'O Pedro fez a reforma do meu banheiro e ficou perfeito! Muito cuidadoso e organizado.', 5, 0),
('55555555-0000-0000-0000-000000000002', '44444444-0000-0000-0000-000000000012', '22222222-0000-0000-0000-000000000001', '11111111-0000-0000-0000-000000000022', '11111111-0000-0000-0000-000000000003', 5, 'Preço justo e entrega rápida da obra. Recomendo muito.', 2, 0),

-- Pintor (1 review)
('55555555-0000-0000-0000-000000000003', '44444444-0000-0000-0000-000000000013', '22222222-0000-0000-0000-000000000002', '11111111-0000-0000-0000-000000000023', '11111111-0000-0000-0000-000000000004', 5, 'Trabalho impecável, não deixou respingos e foi super atencioso.', 3, 0),

-- Eletricista (2 reviews)
('55555555-0000-0000-0000-000000000004', '44444444-0000-0000-0000-000000000014', '22222222-0000-0000-0000-000000000003', '11111111-0000-0000-0000-000000000024', '11111111-0000-0000-0000-000000000001', 5, 'Resolveu um curto-circuito na minha cozinha em tempo recorde! Profissional excelente.', 6, 0),
('55555555-0000-0000-0000-000000000005', '44444444-0000-0000-0000-000000000015', '22222222-0000-0000-0000-000000000003', '11111111-0000-0000-0000-000000000025', '11111111-0000-0000-0000-000000000001', 5, 'Fez a instalação de vários spots de LED e ficou lindo. Muito caprichoso.', 4, 0),

-- Encanador (1 review)
('55555555-0000-0000-0000-000000000006', '44444444-0000-0000-0000-000000000016', '22222222-0000-0000-0000-000000000004', '11111111-0000-0000-0000-000000000026', '11111111-0000-0000-0000-000000000001', 5, 'Excelente serviço, muito pontual e educado.', 4, 0),

-- Mudança (1 review)
('55555555-0000-0000-0000-000000000007', '44444444-0000-0000-0000-000000000017', '22222222-0000-0000-0000-000000000005', '11111111-0000-0000-0000-000000000027', '11111111-0000-0000-0000-000000000005', 4, 'Chegaram no horário combinado e transportaram as caixas sem danos. Bom serviço.', 2, 0),

-- Montador de Móveis (1 review)
('55555555-0000-0000-0000-000000000008', '44444444-0000-0000-0000-000000000018', '22222222-0000-0000-0000-000000000006', '11111111-0000-0000-0000-000000000027', '11111111-0000-0000-0000-000000000006', 5, 'Montou meu guarda-roupas planejado muito rápido. Ficou muito firme, excelente trabalho.', 1, 0),

-- Marido de Aluguel (1 review)
('55555555-0000-0000-0000-000000000009', '44444444-0000-0000-0000-000000000019', '22222222-0000-0000-0000-000000000007', '11111111-0000-0000-0000-000000000026', '11111111-0000-0000-0000-000000000001', 5, 'Instalou cortinas e suportes de TV perfeitamente. Muito caprichoso.', 3, 0),

-- Diarista / Faxina (2 reviews)
('55555555-0000-0000-0000-000000000010', '44444444-0000-0000-0000-000000000020', '22222222-0000-0000-0000-000000000008', '11111111-0000-0000-0000-000000000027', '11111111-0000-0000-0000-000000000002', 5, 'A Juliana é incrível. Casa impecável, cheirosa e organizada. Recomendo de olhos fechados.', 8, 0),
('55555555-0000-0000-0000-000000000011', '44444444-0000-0000-0000-000000000021', '22222222-0000-0000-0000-000000000008', '11111111-0000-0000-0000-000000000025', '11111111-0000-0000-0000-000000000002', 5, 'Fez a faxina pós-obra na minha casa e tirou toda a sujeira difícil de gesso. Nota 10.', 4, 0),

-- Informática / TI (1 review)
('55555555-0000-0000-0000-000000000012', '44444444-0000-0000-0000-000000000022', '22222222-0000-0000-0000-000000000009', '11111111-0000-0000-0000-000000000022', '11111111-0000-0000-0000-000000000007', 4, 'Fez a limpeza física e trocou a pasta térmica do meu notebook gamer, agora não esquenta mais. Bom trabalho.', 3, 0),

-- Aulas de Violão (1 review)
('55555555-0000-0000-0000-000000000013', '44444444-0000-0000-0000-000000000023', '22222222-0000-0000-0000-000000000011', '11111111-0000-0000-0000-000000000026', '11111111-0000-0000-0000-000000000008', 5, 'Professor excelente! Tem muita paciência com iniciantes. Recomendo muito.', 1, 0);

