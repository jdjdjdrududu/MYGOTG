#!/bin/bash

echo "🔍 Диагностика фронтенда Сервис-Крым"
echo "====================================="

BASE_URL="http://localhost:8080/webapp"

# Проверяем основные файлы
echo "📋 Проверяем основные файлы:"
echo ""

echo -n "1. Главная страница (index.html): "
if curl -s -f "$BASE_URL/" > /dev/null; then
    echo "✅ Доступна"
else
    echo "❌ Недоступна"
fi

echo -n "2. CSS файл (unified-styles.css): "
if curl -s -f "$BASE_URL/unified-styles.css" > /dev/null; then
    CSS_SIZE=$(curl -s "$BASE_URL/unified-styles.css" | wc -c)
    echo "✅ Доступен (${CSS_SIZE} байт)"
else
    echo "❌ Недоступен"
fi

echo -n "3. Минимальный тест: "
if curl -s -f "$BASE_URL/minimal-test.html" > /dev/null; then
    echo "✅ Доступен"
else
    echo "❌ Недоступен"
fi

echo -n "4. Тест CSS: "
if curl -s -f "$BASE_URL/test-css.html" > /dev/null; then
    echo "✅ Доступен"
else
    echo "❌ Недоступен"
fi

echo ""
echo "📱 Проверяем JavaScript модули:"
echo ""

JS_FILES=("js/app.js" "js/modules/utils.js" "js/modules/api.js" "js/modules/ui.js" "js/modules/operator-panel.js")

for js_file in "${JS_FILES[@]}"; do
    echo -n "- $js_file: "
    if curl -s -f "$BASE_URL/$js_file" > /dev/null; then
        echo "✅ Доступен"
    else
        echo "❌ Недоступен"
    fi
done

echo ""
echo "🌐 Тестовые URL для браузера:"
echo ""
echo "• Главная страница: $BASE_URL/"
echo "• Минимальный тест: $BASE_URL/minimal-test.html"
echo "• Диагностика CSS: $BASE_URL/test-css.html"
echo "• Диагностика черного экрана: $BASE_URL/debug-black-screen.html"
echo "• Отладка: $BASE_URL/debug.html"

echo ""
echo "💡 Рекомендации:"
echo ""
echo "1. Если видите черный экран с кнопками: $BASE_URL/debug-black-screen.html"
echo "2. Для проверки CSS: $BASE_URL/minimal-test.html"
echo "3. Проверьте консоль браузера (F12) на наличие ошибок"
echo "4. Если CSS не применяется, проверьте Network вкладку в DevTools"
echo "5. Главная страница теперь показывает fallback контент если JS не работает"

echo ""
echo "Диагностика завершена!" 