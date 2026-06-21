INSERT INTO tags (id, name, color, parent_id) VALUES
(1,  'Еда',            '#3C5070', NULL),
(2,  'Здоровье',       '#112250', NULL),
(3,  'Транспорт',      '#E0C68F', NULL),
(4,  'Развлечения',    '#D9CBC2', NULL),
(5,  'Коммунальные',   '#3C5070', NULL),
(6,  'Зарплата',       '#112250', NULL),
(7,  'Фрукты',         '#8A96A9', 1),
(8,  'Кафе',           '#8A96A9', 1),
(9,  'Лекарства',      '#707A96', 2),
(10, 'Врачи',          '#707A96', 2),
(11, 'Такси',          '#ECDCBC', 3),
(12, 'Общественный',   '#ECDCBC', 3)
ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  color = EXCLUDED.color,
  parent_id = EXCLUDED.parent_id;

SELECT setval('tags_id_seq', (SELECT MAX(id) FROM tags));
