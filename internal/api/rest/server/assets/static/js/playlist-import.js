// -------------------------------------------------------------
// Import of external playlists (e.g. YouTube) from the grab form
// -------------------------------------------------------------

import * as notify from './notifications.js';
import { API_PATHS } from './constants.js';

const DIALOG_ID = 'playlist-import-dialog';

const GRAB_FORM_ID = 'grabForm';
const MEDIA_URL_INPUT_ID = 'mediaURLInput';

const LIMIT_DEFAULT = 50;

// URL that points to a playlist (YouTube list=, /playlist, /sets/ etc.).
function isPlaylistURL(rawUrl) {
    const url = (rawUrl || '').trim().toLowerCase();
    if (!url) return false;

    return /[?&]list=/.test(url) || /\/playlist(\?|\/|$)/.test(url) || /\/sets\//.test(url);
}

function isVideoInPlaylistURL(rawUrl) {
    const url = (rawUrl || '').trim().toLowerCase();
    return /[?&]v=/.test(url) || /youtu\.be\//.test(url);
}

function formatDuration(seconds) {
    if (!seconds || seconds <= 0) return '';
    const total = Math.round(seconds);
    const h = Math.floor(total / 3600);
    const m = Math.floor((total % 3600) / 60);
    const s = total % 60;
    const mm = h > 0 ? String(m).padStart(2, '0') : String(m);
    const ss = String(s).padStart(2, '0');
    return h > 0 ? `${h}:${mm}:${ss}` : `${mm}:${ss}`;
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

export function initPlaylistImport() {
    const form = document.getElementById(GRAB_FORM_ID);
    const input = document.getElementById(MEDIA_URL_INPUT_ID);
    const dialog = document.getElementById(DIALOG_ID);

    if (!form || !input || !dialog) return;

    const titleElement = document.getElementById('playlist-import-title');
    const summaryElement = document.getElementById('playlist-import-summary');
    const targetSelect = document.getElementById('playlist-import-target');
    const limitInput = document.getElementById('playlist-import-limit');
    const itemsElement = document.getElementById('playlist-import-items');
    const submitButton = document.getElementById('playlist-import-submit');
    const onlyOneButton = document.getElementById('playlist-import-only-one');

    let currentURL = '';
    let preview = null;
    let skipIntercept = false;

    // grab-форма очищает поле ввода в своём htmx:beforeRequest, поэтому
    // запоминаем последнее введённое значение.
    let lastURL = '';
    input.addEventListener('input', () => {
        lastURL = (input.value || '').trim();
    });

    const closeDialog = () => {
        if (dialog.open) dialog.close();
    };

    const renderPreview = (data) => {
        preview = data;

        titleElement.textContent = data.title || 'Playlist';
        summaryElement.textContent =
            `${data.total} videos in the playlist, ${data.alreadyDownloaded} already downloaded`;

        limitInput.max = String(Math.max(data.items.length, 1));
        limitInput.value = String(Math.min(data.items.length || LIMIT_DEFAULT, data.limit || LIMIT_DEFAULT));

        targetSelect.textContent = '';

        const newOption = document.createElement('option');
        newOption.value = '';
        newOption.textContent = `New playlist: ${data.title || 'Imported playlist'}`;
        targetSelect.appendChild(newOption);

        for (const playlist of data.playlists || []) {
            const option = document.createElement('option');
            option.value = playlist.playlistId;
            option.textContent = playlist.title;
            targetSelect.appendChild(option);
        }

        itemsElement.textContent = '';

        for (const item of data.items || []) {
            const row = document.createElement('div');
            row.className = 'playlist-import-dialog__item';

            const duration = formatDuration(item.duration);
            if (duration) {
                const durationElement = document.createElement('span');
                durationElement.className = 'playlist-import-dialog__item-duration';
                durationElement.textContent = duration;
                row.appendChild(durationElement);
            }

            const title = document.createElement('span');
            title.className = 'playlist-import-dialog__item-title';
            title.textContent = item.title || item.url;
            row.appendChild(title);

            if (item.existing) {
                const badge = document.createElement('span');
                badge.className = 'playlist-dialog__badge playlist-import-dialog__item-badge';
                badge.textContent = 'downloaded';
                row.appendChild(badge);
            }

            itemsElement.appendChild(row);
        }

        onlyOneButton.classList.toggle('hidden', !isVideoInPlaylistURL(currentURL));
    };

    const openDialog = async (url) => {
        currentURL = url;

        summaryElement.textContent = 'Reading the playlist…';
        itemsElement.textContent = '';
        titleElement.textContent = 'Playlist';

        if (typeof dialog.showModal === 'function') {
            dialog.showModal();
        } else {
            dialog.setAttribute('open', 'open');
        }

        const res = await fetch(
            `${API_PATHS.downloaderPlaylists}/preview?url=${encodeURIComponent(url)}&limit=${LIMIT_DEFAULT}`
        );

        if (!res.ok) {
            notify.show(await readErrorMessage(res, 'Failed to read the playlist'), notify.notifyType.ERROR);
            closeDialog();
            return;
        }

        renderPreview(await res.json());
    };

    // Перехватываем отправку формы, если вставили ссылку на плейлист.
    form.addEventListener('htmx:beforeRequest', (event) => {
        if (skipIntercept) {
            skipIntercept = false;
            return;
        }

        const url = (input.value || '').trim() || lastURL;
        if (!isPlaylistURL(url)) return;

        event.preventDefault();
        openDialog(url);
    });

    submitButton.addEventListener('click', async () => {
        if (!preview) return;

        const limit = Math.max(1, Math.min(
            parseInt(limitInput.value || String(LIMIT_DEFAULT), 10) || LIMIT_DEFAULT,
            preview.items.length || LIMIT_DEFAULT,
        ));

        submitButton.disabled = true;

        try {
            const formValue = (name) => form.querySelector(`[name='${name}']`)?.value ?? '';

            const body = {
                url: currentURL,
                playlistId: targetSelect.value || '',
                newPlaylistTitle: targetSelect.value ? '' : (preview.title || ''),
                limit: limit,
                qualityCodec: formValue('quality-codec'),
                qualityResolution: formValue('quality-resolution'),
                format: formValue('format'),
            };

            const res = await fetch(`${API_PATHS.downloaderPlaylists}/import`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(body),
            });

            if (!res.ok) {
                notify.show(await readErrorMessage(res, 'Failed to import the playlist'), notify.notifyType.ERROR);
                return;
            }

            const result = await res.json();

            closeDialog();
            input.value = '';

            const parts = [`Queued: ${result.queued}`];
            if (result.reused) parts.push(`already downloaded: ${result.reused}`);
            if (result.skipped) parts.push(`skipped: ${result.skipped}`);

            notify.show(`Playlist "${result.title}": ${parts.join(', ')}`, notify.notifyType.SUCCESS);
        } finally {
            submitButton.disabled = false;
        }
    });

    onlyOneButton.addEventListener('click', () => {
        closeDialog();

        // Поле ввода могло быть очищено формой — возвращаем исходную ссылку.
        input.value = currentURL;
        lastURL = currentURL;

        skipIntercept = true;
        form.requestSubmit ? form.requestSubmit() : form.submit();
    });
}
