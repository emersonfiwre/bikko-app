-- Seed Data for Testing Bikko Platform (PostgreSQL 16 + PostGIS)

-- 1. Insert Categories
INSERT INTO categories (id, name, icon_url, description, is_active) VALUES
('c1000000-0000-0000-0000-000000000001', 'Pedreiro & Reformas', 'https://pub-bikko.r2.dev/icons/mason.png', 'Serviços de alvenaria, construção e reparos gerais', true),
('c1000000-0000-0000-0000-000000000002', 'Eletricista Residencial', 'https://pub-bikko.r2.dev/icons/electrician.png', 'Instalações elétricas, quadros de luz e curtos', true),
('c1000000-0000-0000-0000-000000000003', 'Encanador & Hidráulica', 'https://pub-bikko.r2.dev/icons/plumber.png', 'Desentupimentos, vazamentos e instalação de pias', true),
('c1000000-0000-0000-0000-000000000004', 'Pintura & Acabamento', 'https://pub-bikko.r2.dev/icons/painter.png', 'Pinturas internas, externas e aplicação de grafiato', true)
ON CONFLICT (name) DO NOTHING;

-- 2. Insert Users (Customers & Bikkers)
INSERT INTO users (id, full_name, email, phone, cpf, password_hash, is_bikker, rating, total_ratings) VALUES
('u1000000-0000-0000-0000-000000000001', 'Cliente Exemplo', 'cliente@bikko.com.br', '(11) 98888-1111', '111.222.333-44', '$2a$10$vN4kY8v6YxW0Nn2/R2R3eeS.p01G5XmZ41sM2Y6Z41sM2Y6Z41sM2', false, 5.00, 1),
('u1000000-0000-0000-0000-000000000002', 'João Silva Eletricista', 'joao.eletrica@bikko.com.br', '(11) 99999-2222', '222.333.444-55', '$2a$10$vN4kY8v6YxW0Nn2/R2R3eeS.p01G5XmZ41sM2Y6Z41sM2Y6Z41sM2', true, 4.90, 15),
('u1000000-0000-0000-0000-000000000003', 'Carlos Pedreiro', 'carlos.obras@bikko.com.br', '(11) 97777-3333', '333.444.555-66', '$2a$10$vN4kY8v6YxW0Nn2/R2R3eeS.p01G5XmZ41sM2Y6Z41sM2Y6Z41sM2', true, 4.80, 8)
ON CONFLICT (email) DO NOTHING;

-- 3. Insert Bikkers (Geospatial Point: São Paulo -23.55052, -46.63330)
INSERT INTO bikkers (id, profession, experience_years, description, location_name, location_geom, rating, total_reviews, is_active) VALUES
('u1000000-0000-0000-0000-000000000002', 'Eletricista Residencial', '8 anos', 'Especialista em instalações de chuveiro, quadros elétricos e iluminação LED', 'Centro, São Paulo - SP', ST_SetSRID(ST_MakePoint(-46.63330, -23.55052), 4326)::geography, 4.90, 15, true),
('u1000000-0000-0000-0000-000000000003', 'Mestre de Obras e Alvenaria', '12 anos', 'Construção, reformas de banheiros, assentamento de pisos e azulejos', 'Pinheiros, São Paulo - SP', ST_SetSRID(ST_MakePoint(-46.68900, -23.56100), 4326)::geography, 4.80, 8, true)
ON CONFLICT (id) DO NOTHING;

-- 4. Insert Services (Data Shaping with Category JOIN)
INSERT INTO services (id, bikker_id, category_id, name, description, thumbnail_url, location_geom, reviews_average, total_reviews, is_active) VALUES
('s1000000-0000-0000-0000-000000000001', 'u1000000-0000-0000-0000-000000000002', 'c1000000-0000-0000-0000-000000000002', 'Troca de Quadro Elétrico & Chuveiros', 'Instalação rápida e segura com emissão de laudo elétrico residencial', 'https://pub-bikko.r2.dev/thumbnails/electrician_1.jpg', ST_SetSRID(ST_MakePoint(-46.63330, -23.55052), 4326)::geography, 4.90, 15, true),
('s1000000-0000-0000-0000-000000000002', 'u1000000-0000-0000-0000-000000000003', 'c1000000-0000-0000-0000-000000000001', 'Reforma de Banheiros & Assentamento de Porcelanato', 'Serviço completo com fino acabamento e garantia de 1 ano', 'https://pub-bikko.r2.dev/thumbnails/mason_1.jpg', ST_SetSRID(ST_MakePoint(-46.68900, -23.56100), 4326)::geography, 4.80, 8, true)
ON CONFLICT (id) DO NOTHING;

-- 5. Insert Completed Order (Ready for testing Rating Cascade)
INSERT INTO quote_requests (id, customer_id, bikker_id, service_id, status, initial_description, location_address, location_geom) VALUES
('q1000000-0000-0000-0000-000000000001', 'u1000000-0000-0000-0000-000000000001', 'u1000000-0000-0000-0000-000000000002', 's1000000-0000-0000-0000-000000000001', 'ACCEPTED', 'Preciso trocar o disjuntor principal da minha casa', 'Av. Paulista, 1000', ST_SetSRID(ST_MakePoint(-46.65588, -23.56141), 4326)::geography)
ON CONFLICT (id) DO NOTHING;

INSERT INTO orders (id, quote_request_id, customer_id, bikker_id, service_id, status, final_price, scheduled_date) VALUES
('o1000000-0000-0000-0000-000000000001', 'q1000000-0000-0000-0000-000000000001', 'u1000000-0000-0000-0000-000000000001', 'u1000000-0000-0000-0000-000000000002', 's1000000-0000-0000-0000-000000000001', 'COMPLETED', 250.00, CURRENT_TIMESTAMP - INTERVAL '1 day')
ON CONFLICT (id) DO NOTHING;
