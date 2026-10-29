-- Explicit development fixture: two accounts, no orders, leads or portfolio inserts.
-- Reapplying preserves work already created by local tests; reset is a separate command.
INSERT INTO users(id,full_name,email,phone,cpf,is_bikker,location_name,location_geom) VALUES
('11111111-1111-1111-1111-111111111111','Lucas Ferreira Mendes','lucas.mendes@bikkofy.com.br','11999990000','345.678.901-22',FALSE,'Pinheiros, São Paulo - SP',ST_SetSRID(ST_MakePoint(-46.6823,-23.5641),4326)::geography),
('22222222-2222-2222-2222-222222222222','Rafael Oliveira Costa','rafael.costa@bikkerfy.com.br','11988880000','456.789.012-33',TRUE,'Pinheiros, São Paulo - SP',ST_SetSRID(ST_MakePoint(-46.6914,-23.5614),4326)::geography)
ON CONFLICT(phone) WHERE deleted_at IS NULL AND phone IS NOT NULL DO UPDATE SET full_name=EXCLUDED.full_name,email=EXCLUDED.email,phone=EXCLUDED.phone,cpf=EXCLUDED.cpf,is_bikker=EXCLUDED.is_bikker,location_name=EXCLUDED.location_name,location_geom=EXCLUDED.location_geom;
INSERT INTO bikkers(id,profession,experience_years,description,location_name,location_geom,is_active)
SELECT id,'Eletricista / Instalação e Manutenção Elétrica','5_TO_10','','Pinheiros, São Paulo - SP',ST_SetSRID(ST_MakePoint(-46.6914,-23.5614),4326)::geography,TRUE FROM users WHERE phone='11988880000' AND deleted_at IS NULL
ON CONFLICT(id) DO UPDATE SET location_name=EXCLUDED.location_name,location_geom=EXCLUDED.location_geom;
