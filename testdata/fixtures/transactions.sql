INSERT INTO transactions (title, amount, date, tag_id, category, specificity, comment, url, file_path, file_name, file_mime_type) VALUES
('Продукты Пятёрочка', 234500, '2026-06-14', 7, 'expense', 'simple', 'Фрукты и вода', NULL, NULL, NULL, NULL),
('Аренда квартиры', 4500000, '2026-06-01', 5, 'expense', 'required', NULL, NULL, NULL, NULL, NULL),
('Зарплата июнь', 15000000, '2026-06-05', 6, 'income', 'simple', NULL, NULL, NULL, NULL, NULL),
('Подписка Netflix', 119900, '2026-06-10', 4, 'expense', 'simple', NULL, 'https://netflix.com/account', NULL, NULL, NULL),
('Врач-терапевт', 350000, '2026-05-20', 10, 'expense', 'required', NULL, NULL, 'uploads/check_001.jpg', 'check_001.jpg', 'image/jpeg'),
('Техосмотр авто', 250000, '2026-04-15', 3, 'expense', 'simple', 'Пройден техосмотр, документы получены', 'https://gibdd.ru', 'uploads/to_doc.pdf', 'техосмотр.pdf', 'application/pdf'),
('Аренда от жильцов', 2500000, '2026-06-03', 6, 'income', 'required', 'Оплата за июнь', NULL, NULL, NULL, NULL),
('Копеечная трата', 1, '2026-06-14', 1, 'expense', 'simple', NULL, NULL, NULL, NULL, NULL),
('Автомобиль', 350000000, '2026-01-15', 3, 'expense', 'simple', NULL, NULL, NULL, NULL, NULL),
('Договор аренды', 100, '2026-06-01', 5, 'expense', 'required', NULL, NULL, 'uploads/contract.docx', 'договор.docx', 'application/vnd.openxmlformats-officedocument.wordprocessingml.document'),
('Яндекс Такси', 45000, '2026-06-13', 11, 'expense', 'simple', NULL, NULL, NULL, NULL, NULL),
('Фриланс-проект', 5000000, '2026-05-31', 6, 'income', 'simple', 'Оплата за сайт', 'https://upwork.com/contracts/123', 'uploads/act.pdf', 'акт_выполненных.pdf', 'application/pdf');
