import * as browser from './browser.js';
import * as common from "./common.js";
import * as dialog from "./dialog.js";
import * as tooltip from './tooltip.js';
import * as notify from './notifications.js';
import { API_PATHS } from './constants.js';

import { initPlaylistsMenus as initMenus } from './pages-list.menu-configs.js';
import { initDomElements } from "./playlist-page.dom.js";

async function readErrorMessage(res, fallback) {
    try {
        const data = await res.json();
        if (data && typeof data === 'object' && 'message' in data && data.message) {
            return data.message;
        }
    } catch (e) {
        // ignore
    }
    return fallback;
}

function initCreateForm() {
    const form = document.getElementById('playlists-create-form');
    if (!form) return;

    const input = document.getElementById('playlists-create-input');
    if (!input) return;

    form.addEventListener('submit', async (event) => {
        event.preventDefault();

        const title = input.value.trim();
        if (!title) {
            input.focus();
            return;
        }

        const button = form.querySelector('button[type="submit"]');
        if (button) button.disabled = true;

        try {
            const res = await fetch(API_PATHS.downloaderPlaylists, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ title }),
            });

            if (!res.ok) {
                notify.show(await readErrorMessage(res, 'Failed to create playlist'), notify.notifyType.ERROR);
                return;
            }

            input.value = '';
            window.location.reload();
        } finally {
            if (button) button.disabled = false;
        }
    });
}

// Фильтр карточек плейлистов по названию/описанию.
// Штатный поиск на этой странице не нужен: он ищет по медиа, а строк media-result
// здесь нет, поэтому перехватываем запрос и фильтруем карточки на месте.
function initPlaylistsSearch() {
    const source = document.getElementById('historySearchInput');
    if (!source) return;

    // Поле поиска в шапке рассчитано на страницы со списком медиа (htmx-post в
    // /downloader/search). Здесь строк медиа нет, поэтому снимаем htmx-атрибуты:
    // элемент заменяем клоном, иначе htmx продолжит дёргать несуществующий поиск.
    const input = source.cloneNode(true);
    for (const attribute of [...input.attributes]) {
        if (attribute.name.startsWith('hx-')) {
            input.removeAttribute(attribute.name);
        }
    }
    source.replaceWith(input);

    const applyFilter = () => {
        const query = input.value.trim().toLowerCase();
        const cards = [...document.querySelectorAll('.playlist-card')];
        let visible = 0;

        for (const card of cards) {
            const text = card.textContent.toLowerCase();
            const match = !query || text.includes(query);
            card.classList.toggle('hidden', !match);
            if (match) visible++;
        }

        if (cards.length === 0) return;

        let empty = document.getElementById('playlists-empty');
        if (!empty) {
            empty = document.createElement('div');
            empty.className = 'playlists__empty hidden';
            empty.id = 'playlists-empty';
            (document.getElementById('playlists') || document.body).appendChild(empty);
        }

        empty.textContent = 'No playlists match the search';
        empty.classList.toggle('hidden', visible > 0);
    };

    input.addEventListener('input', applyFilter);
    input.addEventListener('search', applyFilter);

    const clearButton = document.getElementById('historySearchClearButton');
    if (clearButton) {
        clearButton.addEventListener('click', () => {
            input.value = '';
            applyFilter();
        });
    }
}

// Disable browser scroll position restoration
if ('scrollRestoration' in history) {
    history.scrollRestoration = 'manual';
}

document.addEventListener('DOMContentLoaded', () => {
    initDomElements();

    document.body.classList.add('layout-ready');

    // Initialize common page functionality
    common.initHTMX();

    // Initialize viewport height sync (fixes mobile PWA viewport issues)
    browser.initViewportHeightVar();

    // Initialize header auto-hide on scroll
    common.initHeaderAutoHide();

    // Init dialogs
    dialog.initDialogs();

    // Init tooltips
    tooltip.initTooltips();

    // Init menu
    initMenus();

    // Init the playlist creation form
    initCreateForm();

    // Init playlist cards filter
    initPlaylistsSearch();
});

// Force scroll to top after full page load
window.addEventListener('load', () => {
    window.scrollTo(0, 0);
});
