(() => {
  const sidebar = document.getElementById('sidebar');
  const content = document.getElementById('content');

  function renderMarkdown(md) {
    return typeof marked !== 'undefined' ? marked.parse(md) : md.replace(/\n/g, '<br>');
  }

  function todayStr() {
    const t = new Date();
    return `${t.getFullYear()}-${String(t.getMonth()+1).padStart(2,'0')}-${String(t.getDate()).padStart(2,'0')}`;
  }

  function isPastOrToday(dateStr) {
    return dateStr <= todayStr();
  }

  function formatDate(dateStr) {
    const [y, m, d] = dateStr.split('-').map(Number);
    const dt = new Date(y, m - 1, d);
    return dt.toLocaleDateString('en-US', { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' });
  }

  // --- Calendar ---

  let selectedDate = null;

  function buildCalendarMonth(year, month, daysWithEntries) {
    const entrySet = new Set(daysWithEntries.map(d => d.date));
    const firstDay = new Date(year, month, 1);
    const lastDay = new Date(year, month + 1, 0);
    const startDow = firstDay.getDay();
    const totalDays = lastDay.getDate();
    const today = todayStr();

    const monthNames = ['January','February','March','April','May','June',
      'July','August','September','October','November','December'];

    let html = `<div class="calendar">`;
    html += `<div class="cal-header">`;
    html += `<button class="cal-nav" data-dir="-1">&larr;</button>`;
    html += `<span class="cal-title">${monthNames[month]} ${year}</span>`;
    html += `<button class="cal-nav" data-dir="1">&rarr;</button>`;
    html += `</div>`;
    html += `<div class="cal-grid">`;
    html += ['Sun','Mon','Tue','Wed','Thu','Fri','Sat']
      .map(d => `<div class="cal-dow">${d}</div>`).join('');

    for (let i = 0; i < startDow; i++) {
      html += `<div class="cal-cell empty"></div>`;
    }

    for (let d = 1; d <= totalDays; d++) {
      const dateStr = `${year}-${String(month+1).padStart(2,'0')}-${String(d).padStart(2,'0')}`;
      const hasEntry = entrySet.has(dateStr);
      const past = isPastOrToday(dateStr);
      const isToday = dateStr === today;

      let cls = 'cal-cell';
      if (hasEntry) cls += ' has-entry';
      else if (past) cls += ' past-day';
      if (isToday) cls += ' today';
      if (dateStr === selectedDate) cls += ' active';

      const clickable = hasEntry || past;

      html += `<div class="${cls}" ${clickable ? `data-date="${dateStr}"` : ''}>`;
      html += `<span class="cal-day">${d}</span>`;
      if (hasEntry) html += `<span class="cal-dot"></span>`;
      html += `</div>`;
    }

    html += `</div></div>`;
    return html;
  }

  // --- Days View ---

  let allDays = [];
  let calYear, calMonth;

  async function loadDays() {
    content.innerHTML = '<p class="empty">Loading...</p>';

    const res = await fetch('/api/days');
    allDays = await res.json() || [];

    const now = new Date();
    calYear = now.getFullYear();
    calMonth = now.getMonth();

    renderCalendar();
    content.innerHTML = '<p class="empty">Select a date from the calendar</p>';
  }

  function renderCalendar() {
    let html = buildCalendarMonth(calYear, calMonth, allDays);

    // Action buttons below calendar
    html += `<div class="cal-actions">`;
    html += `<button id="sync-btn" class="action-btn">Sync</button>`;
    html += `<button id="summarize-btn" class="action-btn">Summarize</button>`;
    html += `</div>`;

    sidebar.innerHTML = html;

    sidebar.querySelectorAll('.cal-cell[data-date]').forEach(el => {
      el.addEventListener('click', () => loadDay(el.dataset.date));
    });

    sidebar.querySelectorAll('.cal-nav').forEach(btn => {
      btn.addEventListener('click', () => {
        const dir = parseInt(btn.dataset.dir);
        calMonth += dir;
        if (calMonth < 0) { calMonth = 11; calYear--; }
        if (calMonth > 11) { calMonth = 0; calYear++; }
        renderCalendar();
      });
    });

    document.getElementById('sync-btn').addEventListener('click', runSync);
    document.getElementById('summarize-btn').addEventListener('click', runSummarize);
  }

  async function loadDay(date) {
    selectedDate = date;
    sidebar.querySelectorAll('.cal-cell').forEach(el =>
      el.classList.toggle('active', el.dataset.date === date)
    );
    content.innerHTML = '<p class="empty">Loading...</p>';

    const res = await fetch(`/api/days/${date}`);
    if (!res.ok) {
      content.innerHTML = `<div class="day-header"><h1>${formatDate(date)}</h1></div><p class="empty">No log for this date. Try clicking Sync to pull in session data.</p>`;
      return;
    }

    const data = await res.json();
    let html = `<div class="day-header"><h1>${formatDate(date)}</h1></div>`;

    if (data.summary) {
      const stripped = data.summary.replace(/^#\s+(Daily )?Summary:.*\n+/, '');
      html += `<div class="summary-content">${renderMarkdown(stripped)}</div>`;
    }

    if (data.content) {
      html += `<button class="log-toggle" onclick="this.nextElementSibling.hidden=!this.nextElementSibling.hidden">Show Daily Log</button>`;
      html += `<div class="log-content" hidden>${renderMarkdown(data.content)}</div>`;
    }

    content.innerHTML = html;
  }

  // --- Actions ---

  async function runSync() {
    const date = selectedDate || todayStr();
    const btn = document.getElementById('sync-btn');
    btn.disabled = true;
    btn.textContent = 'Syncing...';
    try {
      const res = await fetch(`/api/sync/${date}`, { method: 'POST' });
      const data = await res.json();
      if (data.ok) {
        btn.textContent = 'Done!';
        // Refresh days list and reload current day
        const daysRes = await fetch('/api/days');
        allDays = await daysRes.json() || [];
        renderCalendar();
        if (selectedDate) loadDay(selectedDate);
      } else {
        btn.textContent = 'Failed';
        console.error(data.output, data.error);
      }
    } catch (e) {
      btn.textContent = 'Error';
      console.error(e);
    }
    setTimeout(() => { btn.textContent = 'Sync'; btn.disabled = false; }, 2000);
  }

  async function runSummarize() {
    const date = selectedDate || todayStr();
    const btn = document.getElementById('summarize-btn');
    btn.disabled = true;
    btn.textContent = 'Summarizing...';
    try {
      const res = await fetch(`/api/summarize/${date}`, { method: 'POST' });
      const data = await res.json();
      if (data.ok) {
        btn.textContent = 'Done!';
        const daysRes = await fetch('/api/days');
        allDays = await daysRes.json() || [];
        renderCalendar();
        if (selectedDate) loadDay(selectedDate);
      } else {
        btn.textContent = 'Failed';
        console.error(data.output, data.error);
      }
    } catch (e) {
      btn.textContent = 'Error';
      console.error(e);
    }
    setTimeout(() => { btn.textContent = 'Summarize'; btn.disabled = false; }, 2000);
  }

  // --- Init ---
  loadDays();
})();
