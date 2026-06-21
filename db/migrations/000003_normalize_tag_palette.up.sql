-- Нормализация цветов меток под палитру finAns
-- Pacific Cyan #3E828E, Muted Teal #A6C9B6, Dark Amethyst #27153D, Powder Blush #F6B6B7

UPDATE tags SET color = CASE id
  WHEN 1 THEN '#3E828E'
  WHEN 2 THEN '#A6C9B6'
  WHEN 3 THEN '#27153D'
  WHEN 4 THEN '#F6B6B7'
  WHEN 5 THEN '#3E828E'
  WHEN 6 THEN '#A6C9B6'
  ELSE color
END
WHERE parent_id IS NULL;

UPDATE tags c
SET color = CASE p.color
  WHEN '#3E828E' THEN '#8BB4BB'
  WHEN '#A6C9B6' THEN '#CAE1D3'
  WHEN '#27153D' THEN '#7D738B'
  WHEN '#F6B6B7' THEN '#FAD3D4'
  ELSE c.color
END
FROM tags p
WHERE c.parent_id = p.id;
