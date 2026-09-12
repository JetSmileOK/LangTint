# LangTint launch kit

Use these only after a real GitHub Release exists and the README contains a genuine short demo recording. Keep the post factual: do not claim universal Windows compatibility, no-warning SmartScreen behavior, or uniqueness that has not been proven.

## Show HN

**Title**

Show HN: LangTint – see your Windows keyboard layout before you type

**Post**

I kept noticing the wrong keyboard layout only after typing a few characters, so I made the layout visible through peripheral vision instead.

LangTint changes the Windows taskbar and the standard Arrow/Hand cursor when English is active; my neutral layout stays visually normal. It does not inspect typed text, guess the intended language, read the clipboard, or permanently poll the current layout.

The current build targets Windows 10 x64. It is open source under MIT and installs for the current user with one Setup executable.

Demo: <README demo link>
Repo / download: https://github.com/JetSmileOK/LangTint

I would especially value reports from multilingual Windows users and feedback on whether per-layout configurable colors would be useful.

## Reddit — r/windows / r/productivity style

**Title**

I made a tiny Windows utility that shows your keyboard layout before you start typing

**Post**

I switch keyboard layouts all day and got tired of discovering the wrong one after typing `ghbdtn`.

So I built LangTint. Instead of another popup or a tiny `ENG/RUS` label, it uses things already in peripheral vision:

- English → light-blue taskbar + light-blue Arrow/Hand cursor
- neutral layout → normal Windows taskbar + normal cursor scheme
- no text correction
- no language guessing
- no clipboard reading
- no permanent layout polling

It currently targets Windows 10 x64 and is MIT-licensed/open source.

Short demo: <GIF link>
GitHub: https://github.com/JetSmileOK/LangTint

I am looking for real-world feedback before broadening Windows/version support. If you use 2–3 layouts, I would also like to know what color behavior would make sense for you.

## Russian / Telegram / Habr-style short post

**Заголовок**

LangTint — раскладку видно ещё до первого символа

**Текст**

Постоянно ловил себя на `ghbdtn`: нужную раскладку замечаешь уже после того, как начал печатать.

Сделал маленькую Windows-утилиту LangTint. Она не исправляет текст и не угадывает язык. Вместо этого раскладка становится заметна боковым зрением:

- EN — светло-голубая панель задач и голубые Arrow/Hand;
- нейтральная раскладка — обычный Windows;
- без оверлея поверх панели;
- без чтения текста и буфера обмена;
- без постоянного опроса раскладки.

Сейчас целевая платформа — Windows 10 x64. Проект открыт под MIT.

Короткая демонстрация: <GIF>
GitHub и установка: https://github.com/JetSmileOK/LangTint

Нужны тесты от людей, которые каждый день работают с несколькими раскладками. Отдельно собираю обратную связь по Windows 11 и настраиваемым цветам.

## Launch order

1. Publish the tested GitHub Release.
2. Add a real 5–10 second RU → EN → RU demo to the first README viewport.
3. Configure GitHub social preview from the same visual identity.
4. Post to one relevant community first; answer every useful comment and fix obvious onboarding problems.
5. Only then post to the next community. Do not cross-post the same message everywhere at once.
6. Link community feedback back to GitHub issues so discussion turns into contributors/testers rather than disappearing in social feeds.

## Do not do

- do not buy stars, upvotes, comments, or fake downloads;
- do not spam unrelated repositories/issues;
- do not claim “first ever” or “the only app”;
- do not hide the unsigned status until real code signing is active;
- do not advertise Windows 11 as supported before real-machine validation;
- do not use fabricated screenshots or testimonials.
