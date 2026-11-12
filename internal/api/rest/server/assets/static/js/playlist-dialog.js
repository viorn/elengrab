import * as notify from './notifications.js';

const DIALOG_ID = 'playlist-dialog';
const SELECTORS = {
    list: '#playlist-dialog-list',
    item: '.playlist-dialog__item',
    checkbox: '.playlist-dialog__checkbox',
    createInput: '#playlist-dialog-create-input',
    createButton: '#playlist-dialog-create-button',
    empty: '#playlist-dialog-empty',
};

function getDialog() {
    const dialog = document.getElementById(DIALOG_ID);
    if (!(dialog instanceof HTMLDialogElement)) return null;
    return dialog;
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

function renderEmptyState(list) {
    if (!list || list.querySelector(SELECTORS.item)) return;
    if (list.querySelector(SELECTORS.empty)) return;

    const empty = document.createElement('div');
    empty.className = 'playlist-dialog__empty';
    empty.id = 'playlist-dialog-empty';
    empty.textContent = 'No playlists yet';
    list.appendChild(empty);
}

function bindCheckbox(checkbox) {
    checkbox.addEventListener('change', async () => {
        const label = checkbox.closest(SELECTORS.item);
        const list = checkbox.closest(SELECTORS.list);
        if (!label || !list) return;

        const playlistId = label.dataset.playlistId;
        const toggleURL = list.dataset.toggleUrl;
        const checked = checkbox.checked;

        if (!playlistId || !toggleURL) return;

        checkbox.disabled = true;

        try {
            const res = await fetch(toggleURL, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ playlistId, checked }),
            });

            if (!res.ok) {
                checkbox.checked = !checked;
                const message = await readErrorMessage(res, 'Failed to update playlist');
                notify.show(message, notify.notifyType.ERROR);
            } else if (checked) {
                notify.show('Added to playlist', notify.notifyType.SUCCESS);
            } else {
                notify.show('Removed from playlist', notify.notifyType.SUCCESS);
            }
        } catch (e) {
            checkbox.checked = !checked;
            notify.show('Failed to update playlist', notify.notifyType.ERROR);
            console.error('Playlist toggle error:', e);
        } finally {
            checkbox.disabled = false;
        }
    });
}

function bindCreate(dialog) {
    const input = dialog.querySelector(SELECTORS.createInput);
    const button = dialog.querySelector(SELECTORS.createButton);
    if (!input || !button) return;

    const create = async () => {
        const title = input.value.trim();
        if (!title) {
            input.focus();
            return;
        }

        const createURL = button.dataset.createUrl;
        if (!createURL) return;

        button.disabled = true;

        try {
            const res = await fetch(createURL, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ title }),
            });

            if (!res.ok) {
                const message = await readErrorMessage(res, 'Failed to create playlist');
                notify.show(message, notify.notifyType.ERROR);
                return;
            }

            input.value = '';

            // Re-render the dialog so the new playlist appears in the list.
            const sourceURL = dialog.dataset.sourceUrl;
            if (sourceURL) {
                await loadPlaylistDialog(dialog, sourceURL, false);
            }
        } catch (e) {
            notify.show('Failed to create playlist', notify.notifyType.ERROR);
            console.error('Playlist create error:', e);
        } finally {
            button.disabled = false;
        }
    };

    button.addEventListener('click', create);
    input.addEventListener('keydown', (event) => {
        if (event.key === 'Enter') {
            event.preventDefault();
            create();
        }
    });
}

function bindDialogContent(dialog) {
    const list = dialog.querySelector(SELECTORS.list);

    if (list) {
        list.querySelectorAll(SELECTORS.checkbox).forEach(bindCheckbox);
        renderEmptyState(list);
    }

    bindCreate(dialog);
}

async function loadPlaylistDialog(dialog, url, show = true) {
    try {
        const res = await fetch(url, {
            method: 'GET',
            headers: { 'Accept': 'text/html' },
        });

        if (!res.ok) {
            const message = await readErrorMessage(res, 'Failed to load playlists');
            notify.show(message, notify.notifyType.ERROR);
            return false;
        }

        dialog.innerHTML = await res.text();
        dialog.dataset.sourceUrl = url;

        bindDialogContent(dialog);

        if (show && !dialog.open) {
            dialog.showModal();
        }

        return true;
    } catch (e) {
        notify.show('Failed to load playlists', notify.notifyType.ERROR);
        console.error('Playlist dialog error:', e);
        return false;
    }
}

/**
 * Opens the "add to playlist" dialog for the given dialog fragment URL.
 * @param {string} url URL returning the playlist dialog HTML fragment.
 */
export async function openPlaylistDialog(url) {
    const dialog = getDialog();
    if (!dialog || !url) return;

    await loadPlaylistDialog(dialog, url);
}
