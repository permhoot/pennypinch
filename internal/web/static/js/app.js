// Copyright © 2026 The Homeport Team
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
(function () {
  'use strict';

  function currency() {
    return document.body.dataset.currency || '$';
  }

  function currencyPosition() {
    return document.body.dataset.currencyPosition || 'left';
  }

  function fmt(v) {
    var c = currency();
    var amount = Number(v).toFixed(2);
    if (currencyPosition() === 'right') {
      return amount + c;
    }
    return c + amount;
  }

  /* ---- Theme ---- */
  function applyTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('theme', theme);
  }

  function initTheme() {
    const toggle = document.getElementById('theme-toggle');
    const stored = localStorage.getItem('theme');
    if (stored) {
      document.documentElement.setAttribute('data-theme', stored);
    } else if (window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches) {
      document.documentElement.setAttribute('data-theme', 'light');
    }
    if (toggle) {
      toggle.addEventListener('click', function () {
        const current = document.documentElement.getAttribute('data-theme') === 'dark' ? 'dark' : 'light';
        applyTheme(current === 'dark' ? 'light' : 'dark');
      });
    }
  }

  /* ---- Toast ---- */
  function showToast(html, type) {
    const toast = document.getElementById('toast');
    if (!toast) return;
    toast.innerHTML = html;
    toast.className = 'toast ' + type;
    toast.hidden = false;
    clearTimeout(toast._timer);
    if (type === 'success') {
      toast._timer = setTimeout(function () { toast.hidden = true; }, 4000);
    }
  }

  function hideToast() {
    const toast = document.getElementById('toast');
    if (toast) toast.hidden = true;
  }

  /* ---- CSV Import ---- */
  function initImport() {
    const btn = document.getElementById('import-btn');
    const fileInput = document.getElementById('import-file');
    if (!btn || !fileInput) return;

    btn.addEventListener('click', function () { fileInput.click(); });

    fileInput.addEventListener('change', async function () {
      const file = fileInput.files && fileInput.files[0];
      if (!file) return;

      const formData = new FormData();
      formData.append('file', file);
      btn.disabled = true;

      try {
        const res = await fetch('/api/import/csv', { method: 'POST', body: formData });
        const data = await res.json();
        if (res.ok) {
          let msg = 'Imported ' + data.imported + ' records';
          if (data.new_categories && data.new_categories.length) {
            msg += '<br>' + data.new_categories.length + ' new categories added';
          }
          showToast(msg, 'success');
          setTimeout(function () { window.location.reload(); }, 900);
        } else {
          showImportErrors(data);
        }
      } catch (err) {
        showToast('Import failed: ' + err.message, 'error');
      } finally {
        fileInput.value = '';
        btn.disabled = false;
      }
    });
  }

  function showImportErrors(data) {
    const toast = document.getElementById('toast');
    if (!toast) return;
    let html = '<strong>Import failed</strong>';
    if (data && data.errors && data.errors.length) {
      html += '<ul>';
      data.errors.slice(0, 20).forEach(function (e) {
        const field = e.field ? '(' + e.field + ') ' : '';
        html += '<li>Line ' + e.line + ' ' + field + e.message + '</li>';
      });
      if (data.errors.length > 20) {
        html += '<li>… and ' + (data.errors.length - 20) + ' more</li>';
      }
      html += '</ul>';
    } else if (data && data.error) {
      html += '<br>' + data.error;
    }
    showToast(html, 'error');
  }

  /* ---- Charts ---- */
  function readJSON(id) {
    const el = document.getElementById(id);
    if (!el) return null;
    try { return JSON.parse(el.textContent); } catch (e) { return null; }
  }

  function initMonthlyChart() {
    const canvas = document.getElementById('monthly-chart');
    const data = readJSON('monthly-data');
    if (!canvas || !data) return;
    const cats = data.expense_categories || [];
    if (!cats.length) return;

    new Chart(canvas, {
      type: 'doughnut',
      data: {
        labels: cats.map(function (c) { return c.category; }),
        datasets: [{
          data: cats.map(function (c) { return Number(c.total); }),
          backgroundColor: cats.map(function (c) { return c.color || '#888'; }),
          borderWidth: 1,
          borderColor: getComputedStyle(document.documentElement).getPropertyValue('--bg').trim() || '#fff'
        }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: { position: 'bottom' },
          tooltip: {
            callbacks: {
              label: function (ctx) {
                const c = cats[ctx.dataIndex];
                return ' ' + c.category + ': ' + fmt(c.total) + ' (' + Number(c.percent).toFixed(1) + '%)';
              }
            }
          }
        }
      }
    });
  }

  function initSankeyChart() {
    const canvas = document.getElementById('sankey-chart');
    const data = readJSON('sankey-data');
    if (!canvas || !data || !window.Chart) return;
    const nodes = data.nodes || [];
    const links = data.links || [];
    if (!nodes.length) return;

    const nodeColor = {};
    const labels = {};
    nodes.forEach(function (n) {
      nodeColor[n.id] = n.color || '#888';
      labels[n.id] = n.label;
    });

    const points = links.map(function (l) {
      return { from: l.from, to: l.to, flow: Number(l.value) };
    });

    new Chart(canvas, {
      type: 'sankey',
      data: {
        datasets: [{
          label: 'Cash flow',
          data: points,
          labels: labels,
          colorFrom: function (c) { return nodeColor[c.raw.from] || '#888'; },
          colorTo: function (c) { return nodeColor[c.raw.to] || '#888'; },
          colorMode: 'gradient',
          borderWidth: 0,
          borderColor: 'transparent',
          color: getComputedStyle(document.documentElement).getPropertyValue('--muted').trim() || '#888'
        }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: { display: false },
          tooltip: {
            callbacks: {
              title: function () { return ''; },
              label: function (ctx) {
                const r = ctx.raw || {};
                return ' ' + (r.from || '') + ' → ' + (r.to || '') + ': ' + fmt(r.flow);
              }
            }
          }
        }
      }
    });
  }

  document.addEventListener('DOMContentLoaded', function () {
    initTheme();
    initImport();

    const toast = document.getElementById('toast');
    if (toast) toast.addEventListener('click', hideToast);

    if (document.getElementById('monthly-chart')) initMonthlyChart();
    if (document.getElementById('sankey-chart')) initSankeyChart();
  });
})();
