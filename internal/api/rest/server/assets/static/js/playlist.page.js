import * as browser from './browser.js';
import * as common from "./common.js";
import * as dialog from "./dialog.js";
import * as tooltip from './tooltip.js';
import * as notify from './notifications.js';
import { initPlayer } from './player.js';
import * as videoPreview from './video-preview.js';
import { API_PATHS } from './constants.js';

import * as sseClient from "./sse.js";
import * as rowEventHandlers from './pages-list.sse.events.js';
import * as commonEventHandlers from './sse.events.js';

import * as playlistPlay from './playlist-play.js';
import { initPlaylistMenus as initMenus } from './pages-list.menu-configs.js';
import * as view from './pages-list.view.js';
import * as search from './pages-list.search.js';
import { CLASS_NAMES, CLASS_SELECTORS, DOM_ELEMENTS, initDomElements } from "./playlist-page.dom.js";

// Global variables
let searchInputClearButton = null;

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

function initPlaylistHeader() {
    const renameButton = document.getElementById('playlist-rename-button');
    const deleteButton = document.getElementById('playlist-delete-button');

    if (renameButton) {
        renameButton.addEventListener('click', async () => {
            const playlistId = renameButton.dataset.playlistId;
            if (!playlistId) return;

            const currentTitle = renameButton.dataset.playlistTitle || '';
            const title = window.prompt('Playlist name', currentTitle);
            if (title === null) return;

            const trimmed = title.trim();
            if (!trimmed || trimmed === currentTitle) return;

            const res = await fetch(`${API_PATHS.downloaderPlaylists}/${playlistId}`, {
                method: 'PATCH',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ title: trimmed }),
            });

            if (!res.ok) {
                notify.show(await readErrorMessage(res, 'Failed to rename playlist'), notify.notifyType.ERROR);
                return;
            }

            const titleElement = document.getElementById('playlist-header-title');
            if (titleElement) {
                titleElement.textContent = trimmed;
            }
            renameButton.dataset.playlistTitle = trimmed;

            notify.show('Playlist renamed', notify.notifyType.SUCCESS);
        });
    }

    if (deleteButton) {
        deleteButton.addEventListener('click', async () => {
            const playlistId = deleteButton.dataset.playlistId;
            if (!playlistId) return;

            if (!window.confirm('Delete this playlist?')) return;

            const res = await fetch(`${API_PATHS.downloaderPlaylists}/${playlistId}`, {
                method: 'DELETE',
            });

            if (!res.ok) {
                notify.show(await readErrorMessage(res, 'Failed to delete playlist'), notify.notifyType.ERROR);
                return;
            }

            window.location.href = API_PATHS.downloaderPlaylists;
        });
    }
}

// Disable browser scroll position restoration
if ('scrollRestoration' in history) {
    history.scrollRestoration = 'manual';
}

document.addEventListener('DOMContentLoaded', () => {
    initDomElements();

    // Apply persisted grid/list layout state on initial page load
    view.initGridView();
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

    // Init playlist header actions
    initPlaylistHeader();

    // Init drag & drop ordering and sequential playback
    playlistPlay.initReorder();
    playlistPlay.initPlayback();

    // Init inline media player
    initPlayer(DOM_ELEMENTS.mediaPlayer, CLASS_NAMES.mediaResultRowThumbnailImageWrapper);

    // Init search elements
    const searching = search.initSearching();
    searchInputClearButton = searching.clearButton;

    // Initialize video preview player
    videoPreview.initVideoPreview();
    videoPreview.initVideoPreviewHover(
        DOM_ELEMENTS.result,
        CLASS_NAMES.mediaResultRow, CLASS_NAMES.mediaResultRowThumbnailImageWrapper
    );
    const refreshVideoPreview = videoPreview.initVideoPreviewScroll(
        DOM_ELEMENTS.result,
        CLASS_NAMES.mediaResultRow, CLASS_NAMES.mediaResultRowThumbnailImageWrapper
    );

    // Lazy-load video thumbnails.
    const thumbnailLazyImages = view.initLazyImages({
        containerSelector: CLASS_SELECTORS.mediaResultThumbnailPlayButton,
        placeholderSelector: CLASS_SELECTORS.mediaResultThumbnailPlaceholder,
    });

    // Lazy-load channel avatars.
    const avatarLazyImages = view.initLazyImages({
        containerSelector: CLASS_SELECTORS.mediaResultAvatar,
    });

    // Initialize view mode bar
    view.initViewModeBar({
        getSearchQueryValues: search.getSearchQueryValues,
        onSuccess: (rows) => {
            refreshVideoPreview(true);
            search.setSearchParametersInUrl();

            const lazyObservers = [
                thumbnailLazyImages,
                avatarLazyImages,
            ];

            for (const observer of lazyObservers) {
                observer.observe(rows);
            }
        },
    });

    // Create SSE connection
    const sseEventHandlers = {
        "row-update": rowEventHandlers.handleRowUpdate,
        "row-patch": rowEventHandlers.handleRowPatch,
        "row-delete": rowEventHandlers.handleRowDelete,
        "row-patch-field": rowEventHandlers.handleRowPatchField,
        "row-start-refreshing": rowEventHandlers.handleRowStartRefreshing,
        "notification": commonEventHandlers.handleNotification,
    };
    sseClient.initSSE(sseEventHandlers);
});

// Force scroll to top after full page load
window.addEventListener('load', () => {
    window.scrollTo(0, 0);
});
