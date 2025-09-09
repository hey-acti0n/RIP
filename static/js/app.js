// Общие утилиты API
async function getJSON(url) {
  const res = await fetch(url, { headers: { Accept: "application/json" } });
  if (!res.ok) throw new Error("HTTP " + res.status);
  return await res.json();
}

function qp(params) {
  const s = new URLSearchParams(params);
  return s.toString();
}

function $(sel, root = document) {
  return root.querySelector(sel);
}
function $all(sel, root = document) {
  return Array.from(root.querySelectorAll(sel));
}

function getRequestId() {
  const el = document.getElementById("requestId");
  return el ? el.value : "1";
}

function showNotification(message, isError = false) {
  // Создаем уведомление
  const notification = document.createElement("div");
  notification.style.cssText = `
    position: fixed;
    top: 20px;
    right: 20px;
    background: ${isError ? "#ff4444" : "#44ff44"};
    color: white;
    padding: 12px 20px;
    border-radius: 8px;
    box-shadow: 0 4px 12px rgba(0,0,0,0.3);
    z-index: 1000;
    font-weight: 500;
    max-width: 300px;
  `;
  notification.textContent = message;
  document.body.appendChild(notification);

  // Удаляем через 3 секунды
  setTimeout(() => {
    if (notification.parentNode) {
      notification.parentNode.removeChild(notification);
    }
  }, 3000);
}

async function refreshCartBadge() {
  const requestId = getRequestId();
  try {
    const data = await getJSON(`/api/cart?${qp({ requestId })}`);
    const count = data.items.reduce(
      (a, b) => a + (b.quantity || b.Quantity || 0),
      0
    );
    const badge = document.getElementById("cartCount");
    if (badge) badge.textContent = String(count);
  } catch (e) {
    console.warn(e);
  }
}

// Каталог
async function initCatalog() {
  const form = document.getElementById("searchForm");
  if (!form) return;
  form.addEventListener("submit", (e) => {
    e.preventDefault();
    const q = $("#q").value.trim();
    const thickness = $("#thickness").value;
    const requestId = getRequestId();
    // Переотрисовываем список карточек — GET /api/services (Network #1)
    loadCards(q, thickness, requestId);
  });
  
  // Добавляем обработчик ввода толщины
  const thicknessInput = $("#thickness");
  if(thicknessInput) {
    thicknessInput.addEventListener("input", (e) => {
      const q = $("#q").value.trim();
      const thickness = e.target.value;
      const requestId = getRequestId();
      loadCards(q, thickness, requestId);
    });
  }
  
  await loadCards($("#q").value.trim(), $("#thickness").value, getRequestId());
  await refreshCartBadge();
}

async function loadCards(q, thickness, requestId) {
  const data = await getJSON(`/api/services?${qp({ q, thickness, requestId })}`);
  // В HTML из Response для списка: ид заявки, кол-во в корзине, url изображений — второй и третий показываются по отдельным GET ниже
  const wrap = document.getElementById("cards");
  wrap.innerHTML = "";
  data.items.forEach((s) => {
    const card = document.createElement("div");
    card.className = "card";
    // Формируем характеристики для отображения
    const props = s.props ? s.props.slice(0, 3).join("<br>") : "";

    card.innerHTML = `
      <div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 15px;">
        <h3 style="margin: 0; flex: 1;">${s.name}</h3>
        <div class="muted" style="font-size: 12px; margin-left: 10px;">Характеристики</div>
      </div>
      <div style="display: flex; justify-content: space-between; align-items: flex-start;">
        <div class="actions" style="flex: 1;">
          <a class="btn" href="/detail/${s.id}">Подробнее</a>
          <a class="btn primary buy" data-id="${s.id}">Купить</a>
        </div>
        <div class="muted" style="font-size: 11px; line-height: 1.3; margin-left: 15px; max-width: 200px; text-align: right;">
          ${props}
        </div>
      </div>`;
    wrap.appendChild(card);
  });
  $all(".buy").forEach((btn) =>
    btn.addEventListener("click", async (e) => {
      e.preventDefault();
      const serviceId = e.currentTarget.getAttribute("data-id");
      try {
        // Добавление в корзину — GET /api/add (Network #2)
        const result = await getJSON(
          `/api/add?${qp({ requestId, serviceId })}`
        );
        if (result.success) {
          showNotification(result.message || "Товар добавлен в корзину");
        } else {
          showNotification(result.error || "Ошибка добавления", true);
        }
        await refreshCartBadge();
      } catch (error) {
        showNotification("Ошибка добавления товара", true);
      }
    })
  );
}

// Детальная страница: кнопка Купить
function initDetail() {
  const buy = document.getElementById("buyBtn");
  if (!buy) return;
  buy.addEventListener("click", async (e) => {
    e.preventDefault();
    const serviceId = buy.getAttribute("data-id");
    const requestId = "1";
    try {
      const result = await getJSON(`/api/add?${qp({ requestId, serviceId })}`);
      if (result.success) {
        showNotification(result.message || "Товар добавлен в корзину");
        window.location.href = `/calc?${qp({ requestId })}`;
      } else {
        showNotification(result.error || "Ошибка добавления", true);
      }
    } catch (error) {
      showNotification("Ошибка добавления товара", true);
    }
  });
}

// Страница расчёта
function initCalc() {
  const form = document.getElementById("calcForm");
  if (!form) {
    return;
  }

  // Кнопка очистки корзины
  const clearBtn = document.getElementById("clearCartBtn");
  if (clearBtn) {
    clearBtn.addEventListener("click", async (e) => {
      e.preventDefault();
      const requestId = getRequestId();
      try {
        const result = await getJSON(`/api/clear?${qp({ requestId })}`);
        if (result.success) {
          showNotification(result.message || "Корзина очищена");
          await refreshCartBadge();
          // Очищаем результаты и перезагружаем товары
          await loadCartItems();
        } else {
          showNotification("Ошибка очистки корзины", true);
        }
      } catch (error) {
        showNotification("Ошибка очистки корзины", true);
      }
    });
  }

  // Загружаем товары в корзине при загрузке страницы (с задержкой)
  setTimeout(() => {
    if (window.location.pathname === '/calc') {
      loadCartItems();
    }
  }, 100);

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    const requestId = getRequestId();
    const mass = $("#mass").value;
    const frequency = $("#frequency").value;
    
    if (!mass || !frequency) {
      showNotification("Заполните все поля для расчета", true);
      return;
    }
    
    // Выполняем расчёт — GET /api/calc (Network #3)
    try {
      const data = await getJSON(
        `/api/calc?${qp({ requestId, mass, frequency })}`
      );
      const box = document.getElementById("results");
      box.innerHTML = "";
      
      if (data.results.length === 0) {
        box.innerHTML = '<div class="muted">Добавьте товары в корзину для расчета</div>';
        return;
      }
      
      data.results.forEach((r) => {
        const el = document.createElement("div");
        el.className = "result-card";
        el.innerHTML = `<div class="muted">${r.serviceName}</div>
          <div>Собственная частота: <b>${r.naturalHz} Гц</b></div>
          <div>Ожидаемая изоляция: <b>${r.isolationPercent}%</b></div>`;
        box.appendChild(el);
      });
    } catch (error) {
      showNotification("Ошибка расчета", true);
    }
  });
}

// Функция для загрузки товаров в корзине
async function loadCartItems() {
  const requestId = getRequestId();
  const box = document.getElementById("results");
  
  if (!box) {
    return;
  }
  
  try {
    const cartData = await getJSON(`/api/cart?${qp({ requestId })}`);
    
    if (!cartData.items || cartData.items.length === 0) {
      box.innerHTML = '<div class="muted">Корзина пуста. Добавьте товары для расчета.</div>';
      return;
    }
    
    box.innerHTML = "";
    
    // Загружаем информацию о товарах
    let servicesMap = {};
    try {
      const servicesData = await getJSON(`/api/services?${qp({ requestId })}`);
      servicesData.items.forEach(service => {
        servicesMap[service.id] = service;
      });
    } catch (error) {
      box.innerHTML = '<div class="muted">Ошибка загрузки информации о товарах</div>';
      return;
    }
    
    cartData.items.forEach((item) => {
      const service = servicesMap[item.serviceId];
      if (service) {
        const props = service.props ? service.props.slice(0, 3).join('<br>') : '';
        const el = document.createElement("div");
        el.className = "result-card";
        el.innerHTML = `
          <div style="display: flex; justify-content: space-between; align-items: flex-start;">
            <div style="flex: 1;">
              <div style="font-weight: 600; margin-bottom: 8px;">${service.name}</div>
            </div>
            <div class="muted" style="font-size: 11px; line-height: 1.3; margin-left: 15px; max-width: 200px; text-align: right;">
              ${props}
            </div>
          </div>
        `;
        box.appendChild(el);
      }
    });
  } catch (error) {
    console.error("Ошибка загрузки корзины:", error);
  }
}

document.addEventListener("DOMContentLoaded", () => {
  const path = window.location.pathname;
  
  if (path === '/' || path === '/catalog') {
    initCatalog();
  } else if (path.startsWith('/detail/')) {
    initDetail();
  } else if (path === '/calc') {
    initCalc();
  }
});
