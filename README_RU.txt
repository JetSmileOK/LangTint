TaskbarLayoutTint v1.5.3 — COMPOSITION + ARROW/HAND CURSOR — Windows 10 x64
===========================================================================

BUILD_ID:
TaskbarLayoutTint_COMPOSITION_CURSOR_v1.5.3_20260911

Что исправлено относительно v1.5.2
----------------------------------
В v1.5.2 перекрашивались 13 системных типов курсора. Тонкий I-Beam над текстом
и некоторые служебные курсоры на светлом/сером фоне могли визуально исчезать.

v1.5.3 перекрашивает ТОЛЬКО:
  - OCR_NORMAL: обычная стрелка;
  - OCR_HAND: рука над ссылками/кнопками.

Остаются полностью штатными Windows:
  I-Beam, Wait, Cross, Up, все Resize, Move, No, AppStarting и остальные.

Перед применением EN программа всегда вызывает SPI_SETCURSORS. Поэтому остатки
полностью окрашенного набора v1.5.2 сначала возвращаются к пользовательской
схеме Windows, и только после этого Arrow + Hand становятся голубыми.

Поведение
---------
RU:
  - штатная панель Windows;
  - вся штатная схема курсоров.

EN:
  - direct taskbar composition #B7E9FF;
  - Arrow + Hand: #B7E9FF с темным контуром;
  - I-Beam и служебные курсоры: штатные Windows.

Permanent polling: ZERO.
Mouse hook: NONE.
Overlay window: NONE.

Установка
---------
Можно запустить:
  RUN_ALL_AND_INSTALL.cmd

или двойным кликом:
  TaskbarLayoutTint_v1_5_3.exe

До установки v1.5.3 автоматически останавливает и удаляет установленные
предыдущие семейства, включая v1.5.2, восстанавливает штатную панель и полный
набор курсоров, затем выполняет acceptance + CPU test.

Отчет:
  %USERPROFILE%\Downloads\TaskbarLayoutTint_v1_5_3_REPORT.txt

Исходный код находится в папке source.
