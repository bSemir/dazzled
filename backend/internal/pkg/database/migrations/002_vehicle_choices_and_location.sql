-- Expand vehicle options and add optional reporter-entered location text.
CREATE TABLE incidents_new (
    id INTEGER PRIMARY KEY,
    vehicle_type TEXT NOT NULL CHECK(vehicle_type IN ('Hatchback','Sedan','SUV / crossover','Coupe','Wagon / estate','Pickup truck','Van','Minivan','Bus','Motorcycle','Scooter','Bicycle','Other')),
    brand TEXT NOT NULL CHECK(brand IN ('BMW','Audi','Mercedes-Benz','Tesla','Ford','Volkswagen','Toyota','Honda','Hyundai','Kia','Nissan','Mazda','Subaru','Volvo','Škoda','Peugeot','Renault','Opel','Dacia','Jeep','Porsche','Lexus','Land Rover','Other','Unknown','Aftermarket ''xenon'' bulbs from a webshop')),
    location TEXT NOT NULL DEFAULT '' CHECK(length(location) <= 100),
    severity INTEGER NOT NULL CHECK(severity BETWEEN 1 AND 5),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

INSERT INTO incidents_new(id, vehicle_type, brand, location, severity, created_at)
SELECT id,
       CASE vehicle_type
           WHEN 'SUV' THEN 'SUV / crossover'
           WHEN 'Pickup / truck' THEN 'Pickup truck'
           WHEN 'Motorbike' THEN 'Motorcycle'
           ELSE vehicle_type
       END,
       brand, '', severity, created_at
FROM incidents;

DROP TABLE incidents;
ALTER TABLE incidents_new RENAME TO incidents;

CREATE INDEX incidents_created_at_idx ON incidents(created_at);
CREATE INDEX incidents_brand_created_idx ON incidents(brand, created_at);
CREATE INDEX incidents_vehicle_created_idx ON incidents(vehicle_type, created_at);
