// -------------------------------------------------------------
// Playlist page: manual ordering (drag & drop) and sequential playback
// -------------------------------------------------------------

import * as notify from './notifications.js';
import { API_PATHS } from './constants.js';

const ROWS_CONTAINER_ID = 'media-result-items';
const ROWS_ID = 'media-result-rows';
const ROW_SELECTOR = '.media-result__row';
const PLAY_BUTTON_SELECTOR = '.media-play-button';

const HANDLE_CLASS = 'playlist-row-drag-handle';
const DRAGGING_CLASS = 'playlist-row--dragging';

const MODE_ORDER = 'order';
const MODE_REPEAT = 'repeat';
const MODE_SHUFFLE = 'shuffle';

const HANDLE_SVG = '<svg width="1em" height="1em" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">'
    + '<circle cx="9" cy="6" r="1.6"/><circle cx="15" cy="6" r="1.6"/>'
    + '<circle cx="9" cy="12" r="1.6"/><circle cx="15" cy="12" r="1.6"/>'
    + '<circle cx="9" cy="18" r="1.6"/><circle cx="15" cy="18" r="1.6"/></svg>';

function getRowsContainer() {
    return document.getElementById(ROWS_CONTAINER_ID);
}

function getPlayableRows() {
    const container = getRowsContainer();
    if (!container) return [];

    return [...container.querySelectorAll(ROW_SELECTOR)]
        .filter((row) => row.dataset.media && row.dataset.itemId);
}

function getPlaylistId() {
    const rows = document.getElementById(ROWS_ID);
    if (!rows || !rows.dataset.searchFiltersJson) return '';

    try {
        return JSON.parse(rows.dataset.searchFiltersJson).playlistId || '';
    } catch (e) {
        return '';
    }
}

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

// -------------------------------------------------------------
// Drag & drop ordering
// -------------------------------------------------------------

export function initReorder() {
    const container = getRowsContainer();
    if (!container) return;

    const playlistId = getPlaylistId();
    if (!playlistId) return;

    // Кнопка Rename рендерится только тем, кто может менять плейлист.
    if (!document.getElementById('playlist-rename-button')) return;

    let draggedRow = null;
    let orderBefore = '';

    const currentOrder = () => [...container.querySelectorAll(ROW_SELECTOR)]
        .filter((row) => row.dataset.itemId)
        .map((row) => row.dataset.itemId);

    const saveOrder = async (ids) => {
        const res = await fetch(`${API_PATHS.downloaderPlaylists}/${playlistId}/reorder`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ids }),
        });

        if (!res.ok) {
            notify.show(await readErrorMessage(res, 'Failed to save the order'), notify.notifyType.ERROR);
            window.location.reload();
            return;
        }

        notify.show('Order saved', notify.notifyType.SUCCESS);
    };

    const attachHandle = (row) => {
        if (!row.dataset.itemId || row.querySelector(`.${HANDLE_CLASS}`)) return;

        const handle = document.createElement('span');
        handle.className = HANDLE_CLASS;
        handle.draggable = true;
        handle.title = 'Drag to reorder';
        handle.setAttribute('aria-label', 'Drag to reorder');
        handle.innerHTML = HANDLE_SVG;

        handle.addEventListener('dragstart', (event) => {
            draggedRow = row;
            orderBefore = currentOrder().join(',');
            row.classList.add(DRAGGING_CLASS);
            event.dataTransfer.effectAllowed = 'move';
            event.dataTransfer.setData('text/plain', row.dataset.itemId);
        });

        handle.addEventListener('dragend', async () => {
            row.classList.remove(DRAGGING_CLASS);
            draggedRow = null;

            const orderAfter = currentOrder();
            if (orderAfter.join(',') === orderBefore) return;

            await saveOrder(orderAfter);
        });

        row.prepend(handle);
    };

    const attachHandles = () => {
        for (const row of container.querySelectorAll(ROW_SELECTOR)) {
            attachHandle(row);
        }
    };

    container.addEventListener('dragover', (event) => {
        if (!draggedRow) return;

        event.preventDefault();
        event.dataTransfer.dropEffect = 'move';

        const target = event.target.closest(ROW_SELECTOR);
        if (!target || target === draggedRow || !container.contains(target)) return;

        const rect = target.getBoundingClientRect();
        const isGrid = rect.width < container.clientWidth * 0.6;

        const after = isGrid
            ? (event.clientX - rect.left) > rect.width / 2
            : (event.clientY - rect.top) > rect.height / 2;

        target.parentNode.insertBefore(draggedRow, after ? target.nextSibling : target);
    });

    container.addEventListener('drop', (event) => {
        if (!draggedRow) return;
        event.preventDefault();
    });

    attachHandles();

    // Строки перерисовываются SSE/htmx — возвращаем ручки на место.
    const observer = new MutationObserver(() => attachHandles());
    observer.observe(container, { childList: true });
}

// -------------------------------------------------------------
// Sequential playback: in order / repeat / shuffle
// -------------------------------------------------------------

const queue = {
    rows: [],
    index: -1,
    mode: MODE_ORDER,
    active: false,
};

function getMediaElements() {
    return [
        document.querySelector('#media-player-video video'),
        document.querySelector('#media-player-audio audio'),
    ].filter(Boolean);
}

function playRowAt(index) {
    const row = queue.rows[index];
    if (!row) return;

    const playButton = row.querySelector(PLAY_BUTTON_SELECTOR);
    if (!playButton) return;

    queue.index = index;
    playButton.click();
    row.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
}

function nextIndex() {
    const total = queue.rows.length;
    if (total === 0) return -1;

    if (queue.mode === MODE_SHUFFLE) {
        if (total === 1) return 0;

        let index;
        do {
            index = Math.floor(Math.random() * total);
        } while (index === queue.index);

        return index;
    }

    const next = queue.index + 1;
    if (next < total) return next;

    return queue.mode === MODE_REPEAT ? 0 : -1;
}

export function initPlayback() {
    const playAllButton = document.getElementById('playlist-play-all-button');
    const modeSelect = document.getElementById('playlist-play-mode');

    if (!playAllButton) return;

    const handleEnded = () => {
        if (!queue.active) return;

        // Плеер закрыли — очередь больше не нужна.
        if (!document.body.classList.contains('media-player')) {
            queue.active = false;
            return;
        }

        const next = nextIndex();
        if (next < 0) {
            queue.active = false;
            return;
        }

        playRowAt(next);
    };

    for (const element of getMediaElements()) {
        element.addEventListener('ended', handleEnded);
    }

    playAllButton.addEventListener('click', () => {
        const rows = getPlayableRows();
        if (rows.length === 0) {
            notify.show('Playlist is empty', notify.notifyType.ERROR);
            return;
        }

        queue.rows = rows;
        queue.index = -1;
        queue.mode = modeSelect ? modeSelect.value : MODE_ORDER;
        queue.active = true;

        playRowAt(0);
    });

    if (modeSelect) {
        modeSelect.addEventListener('change', () => {
            queue.mode = modeSelect.value;
        });
    }
}
