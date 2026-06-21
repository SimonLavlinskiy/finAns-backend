-- Палитра luxury: Sapphire #3C5070, Royal Blue #112250, Quicksand #E0C68F, Shellstone #D9CBC2

UPDATE tags SET color = CASE id
  WHEN 1 THEN '#3C5070'
  WHEN 2 THEN '#112250'
  WHEN 3 THEN '#E0C68F'
  WHEN 4 THEN '#D9CBC2'
  WHEN 5 THEN '#3C5070'
  WHEN 6 THEN '#112250'
  ELSE color
END
WHERE parent_id IS NULL;

UPDATE tags c
SET color = CASE p.color
  WHEN '#112250' THEN '#707A96'
  WHEN '#3C5070' THEN '#8A96A9'
  WHEN '#E0C68F' THEN '#ECDCBC'
  WHEN '#D9CBC2' THEN '#E8E0DA'
  -- legacy palette → new light tints
  WHEN '#3E828E' THEN '#8A96A9'
  WHEN '#A6C9B6' THEN '#E8E0DA'
  WHEN '#27153D' THEN '#707A96'
  WHEN '#F6B6B7' THEN '#ECDCBC'
  ELSE c.color
END
FROM tags p
WHERE c.parent_id = p.id;

-- корневые метки со старыми цветами → новая палитра по кругу
UPDATE tags SET color = CASE (id % 4)
  WHEN 1 THEN '#112250'
  WHEN 2 THEN '#3C5070'
  WHEN 3 THEN '#E0C68F'
  ELSE '#D9CBC2'
END
WHERE parent_id IS NULL
  AND color IN (
    '#3E828E', '#A6C9B6', '#27153D', '#F6B6B7',
    '#22C55E', '#EF4444', '#6C63FF', '#F59E0B', '#94A3B8',
    '#8BB4BB', '#CAE1D3', '#7D738B', '#FAD3D4'
  );
