import type { Elements } from '../../app/dom';
import { selectedRequest, type AppState } from '../../entities/request/model/request-state';
import type { RequestTab, WebhookRequest } from '../../entities/request/model/types';
import {
  formatBytes,
  formatDate,
  isBase64,
  looksLikeJSON,
  methodClass,
  syntaxHighlightJSON,
} from '../../shared/lib/format';
import {
  fetchHookResponse,
  saveHookResponse,
  fetchNotification,
  saveNotification,
} from '../../features/endpoint-session/api/endpoint-api';

export function renderSelectedDetail(elements: Elements, state: AppState): void {
  const request = selectedRequest(state);
  if (!request) {
    showPlaceholder(elements);
    return;
  }

  elements.detailPlaceholder.classList.add('hidden');
  elements.detailContent.classList.remove('hidden');

  elements.detailMethod.textContent = request.method;
  elements.detailMethod.className = `method-badge ${methodClass(request.method)}`;
  elements.detailPath.textContent = request.path;
  elements.detailTimestamp.textContent = formatDate(request.receivedAt);

  renderActiveTab(elements, request, state.activeTab, state.token);
  renderTabButtons(elements, state.activeTab);
}

export function showPlaceholder(elements: Elements): void {
  elements.detailPlaceholder.classList.remove('hidden');
  elements.detailContent.classList.add('hidden');
}

function renderActiveTab(
  elements: Elements,
  request: WebhookRequest,
  tab: RequestTab,
  token: string | null,
): void {
  elements.tabContent.replaceChildren();

  switch (tab) {
    case 'headers':
      elements.tabContent.appendChild(createKeyValueTable(parseHeaders(request.headers)));
      break;
    case 'body':
      elements.tabContent.appendChild(createBody(request.body, request.contentType));
      break;
    case 'query':
      elements.tabContent.appendChild(createKeyValueTable(parseQuery(request.query)));
      break;
    case 'info':
      elements.tabContent.appendChild(createInfo(request));
      break;
    case 'settings':
      elements.tabContent.appendChild(createSettingsForm(token));
      break;
  }
}

function renderTabButtons(elements: Elements, activeTab: RequestTab): void {
  for (const button of elements.tabButtons) {
    button.classList.toggle('active', button.dataset.tab === activeTab);
  }
}

function createKeyValueTable(pairs: Array<[string, string]>): HTMLElement {
  if (pairs.length === 0) {
    const message = document.createElement('p');
    message.className = 'empty-table-message';
    message.textContent = 'No entries';
    return message;
  }

  const table = document.createElement('table');
  table.className = 'kv-table';

  const thead = document.createElement('thead');
  const headerRow = document.createElement('tr');
  const keyHeader = document.createElement('th');
  keyHeader.textContent = 'Key';
  const valueHeader = document.createElement('th');
  valueHeader.textContent = 'Value';
  headerRow.append(keyHeader, valueHeader);
  thead.appendChild(headerRow);

  const tbody = document.createElement('tbody');
  for (const [key, value] of pairs) {
    const row = document.createElement('tr');
    const keyCell = document.createElement('td');
    keyCell.textContent = key;
    const valueCell = document.createElement('td');
    valueCell.textContent = value;
    row.append(keyCell, valueCell);
    tbody.appendChild(row);
  }

  table.append(thead, tbody);
  return table;
}

function createBody(body: unknown, contentType: string | undefined): HTMLPreElement {
  const pre = document.createElement('pre');
  pre.className = 'body-pre';

  if (!body || (typeof body === 'string' && body.trim() === '') || isEmptyArray(body)) {
    pre.classList.add('body-empty');
    pre.textContent = '(empty body)';
    return pre;
  }

  let text: string;
  if (typeof body === 'string') {
    text = body;
  } else {
    text = String(body);
  }

  if (isBase64(text)) {
    try {
      text = atob(text);
    } catch {
      // Keep original body text if base64 decoding fails.
    }
  }

  const contentTypeValue = (contentType || '').toLowerCase();
  if (contentTypeValue.includes('json') || looksLikeJSON(text)) {
    try {
      const parsed = JSON.parse(text) as unknown;
      // syntaxHighlightJSON escapes every token before wrapping it in controlled span markup.
      pre.innerHTML = syntaxHighlightJSON(JSON.stringify(parsed, null, 2));
      return pre;
    } catch {
      // Fall through to plain text rendering.
    }
  }

  pre.textContent = text;
  return pre;
}

function createInfo(request: WebhookRequest): HTMLDivElement {
  const size = request.bodySize != null ? formatBytes(request.bodySize) : '0 B';
  const grid = document.createElement('div');
  grid.className = 'info-grid';
  grid.append(
    createInfoCard('Remote IP', request.remoteAddr || '—'),
    createInfoCard('Content-Type', request.contentType || '—'),
    createInfoCard('Body Size', size),
    createInfoCard('Received At', formatDate(request.receivedAt)),
    createInfoCard('Hook ID', String(request.hookId || '—')),
    createInfoCard('Request ID', String(request.id || '—')),
  );
  return grid;
}

function createInfoCard(label: string, value: string): HTMLDivElement {
  const card = document.createElement('div');
  card.className = 'info-card';

  const cardLabel = document.createElement('div');
  cardLabel.className = 'card-label';
  cardLabel.textContent = label;

  const cardValue = document.createElement('div');
  cardValue.className = 'card-value';
  cardValue.textContent = value;

  card.append(cardLabel, cardValue);
  return card;
}

function parseHeaders(raw: string | undefined): Array<[string, string]> {
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>;
    const pairs: Array<[string, string]> = [];
    for (const [key, value] of Object.entries(parsed)) {
      if (Array.isArray(value)) value.forEach((item) => pairs.push([key, String(item)]));
      else pairs.push([key, String(value)]);
    }
    return pairs;
  } catch {
    return [];
  }
}

function parseQuery(raw: string | undefined): Array<[string, string]> {
  if (!raw) return [];
  try {
    return Array.from(new URLSearchParams(raw).entries());
  } catch {
    return [];
  }
}

function isEmptyArray(value: unknown): boolean {
  return Array.isArray(value) && value.length === 0;
}

function createSettingsForm(token: string | null): HTMLDivElement {
  const wrap = document.createElement('div');
  wrap.className = 'settings-form';

  const statusLabel = document.createElement('label');
  statusLabel.textContent = 'Response Status Code';
  const statusInput = document.createElement('input');
  statusInput.type = 'number';
  statusInput.min = '100';
  statusInput.max = '599';
  statusInput.value = '200';
  statusInput.className = 'settings-input';

  const headersLabel = document.createElement('label');
  headersLabel.textContent = 'Response Headers (JSON)';
  const headersInput = document.createElement('textarea');
  headersInput.className = 'settings-textarea';
  headersInput.placeholder = '{"Content-Type": "application/json"}';
  headersInput.rows = 3;

  const bodyLabel = document.createElement('label');
  bodyLabel.textContent = 'Response Body';
  const bodyInput = document.createElement('textarea');
  bodyInput.className = 'settings-textarea';
  bodyInput.placeholder = '{"ok": true}';
  bodyInput.rows = 5;

  const saveBtn = document.createElement('button');
  saveBtn.className = 'settings-save-btn';
  saveBtn.textContent = 'Save';

  // Telegram notifications section
  const divider = document.createElement('hr');
  divider.className = 'settings-divider';

  const tgTitle = document.createElement('h4');
  tgTitle.className = 'settings-section-title';
  tgTitle.textContent = 'Telegram Notifications';

  const tgTokenLabel = document.createElement('label');
  tgTokenLabel.textContent = 'Bot Token';
  const tgTokenInput = document.createElement('input');
  tgTokenInput.type = 'text';
  tgTokenInput.className = 'settings-input';
  tgTokenInput.placeholder = '123456:ABC-DEF...';

  const tgChatLabel = document.createElement('label');
  tgChatLabel.textContent = 'Chat ID';
  const tgChatInput = document.createElement('input');
  tgChatInput.type = 'text';
  tgChatInput.className = 'settings-input';
  tgChatInput.placeholder = '-1001234567890';

  const tgProxyLabel = document.createElement('label');
  tgProxyLabel.textContent = 'Proxy URL (optional)';
  const tgProxyInput = document.createElement('input');
  tgProxyInput.type = 'text';
  tgProxyInput.className = 'settings-input';
  tgProxyInput.placeholder = 'socks5://127.0.0.1:1080';

  const tgSaveBtn = document.createElement('button');
  tgSaveBtn.className = 'settings-save-btn';
  tgSaveBtn.textContent = 'Save Notifications';

  wrap.append(
    statusLabel,
    statusInput,
    headersLabel,
    headersInput,
    bodyLabel,
    bodyInput,
    saveBtn,
    divider,
    tgTitle,
    tgTokenLabel,
    tgTokenInput,
    tgChatLabel,
    tgChatInput,
    tgProxyLabel,
    tgProxyInput,
    tgSaveBtn,
  );

  if (token) {
    void fetchHookResponse(token).then((resp) => {
      statusInput.value = String(resp.statusCode || 200);
      headersInput.value = Object.keys(resp.headers || {}).length
        ? JSON.stringify(resp.headers, null, 2)
        : '';
      bodyInput.value = resp.body || '';
    });

    void fetchNotification(token).then((n) => {
      tgTokenInput.value = n.telegramBotToken;
      tgChatInput.value = n.telegramChatId;
      tgProxyInput.value = n.proxyUrl;
    });

    saveBtn.addEventListener('click', () => {
      let headers: Record<string, string> = {};
      try {
        if (headersInput.value.trim()) {
          headers = JSON.parse(headersInput.value) as Record<string, string>;
        }
      } catch {
        saveBtn.textContent = 'Invalid JSON in headers';
        setTimeout(() => (saveBtn.textContent = 'Save'), 2000);
        return;
      }

      saveBtn.disabled = true;
      void saveHookResponse(token, {
        statusCode: parseInt(statusInput.value, 10) || 200,
        headers,
        body: bodyInput.value,
      })
        .then(() => {
          saveBtn.textContent = 'Saved!';
          setTimeout(() => (saveBtn.textContent = 'Save'), 2000);
        })
        .catch(() => {
          saveBtn.textContent = 'Error';
          setTimeout(() => (saveBtn.textContent = 'Save'), 2000);
        })
        .finally(() => {
          saveBtn.disabled = false;
        });
    });

    tgSaveBtn.addEventListener('click', () => {
      tgSaveBtn.disabled = true;
      void saveNotification(token, {
        telegramBotToken: tgTokenInput.value.trim(),
        telegramChatId: tgChatInput.value.trim(),
        proxyUrl: tgProxyInput.value.trim(),
      })
        .then(() => {
          tgSaveBtn.textContent = 'Saved!';
          setTimeout(() => (tgSaveBtn.textContent = 'Save Notifications'), 2000);
        })
        .catch(() => {
          tgSaveBtn.textContent = 'Error';
          setTimeout(() => (tgSaveBtn.textContent = 'Save Notifications'), 2000);
        })
        .finally(() => {
          tgSaveBtn.disabled = false;
        });
    });
  }

  return wrap;
}
