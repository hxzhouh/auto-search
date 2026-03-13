package webui

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"auto-search/internal/content"
	"auto-search/internal/query"
)

type Server struct {
	repo      *content.Repository
	queryRepo *query.Repository
}

type cleanedItemResponse struct {
	ID               int64              `json:"id"`
	QueryText        string             `json:"query_text"`
	RSSTitle         string             `json:"rss_title"`
	RSSSourceSite    string             `json:"rss_source_site"`
	RSSPublishedAt   string             `json:"rss_published_at"`
	FinalURL         string             `json:"final_url"`
	CanonicalURL     string             `json:"canonical_url"`
	CleanedTitle     string             `json:"cleaned_title"`
	CleanedSummary   string             `json:"cleaned_summary"`
	CleanedContent   string             `json:"cleaned_content"`
	Language         string             `json:"language"`
	ContentType      string             `json:"content_type"`
	QualityScore     int                `json:"quality_score"`
	ImportanceScore  int                `json:"importance_score"`
	WriteworthyScore int                `json:"writeworthy_score"`
	IsRelevant       bool               `json:"is_relevant"`
	AngleHint        string             `json:"angle_hint"`
	AIReason         string             `json:"ai_reason"`
	UpdatedAt        string             `json:"updated_at"`
	Tags             []content.TagInput `json:"tags"`
}

type cleanedListResponse struct {
	Source string                `json:"source"`
	Limit  int                   `json:"limit"`
	Count  int                   `json:"count"`
	Items  []cleanedItemResponse `json:"items"`
}

var pageTemplate = template.Must(template.New("cleaned").Parse(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Auto Search 素材库</title>
  <style>
    :root {
      --bg: #f6f2e8;
      --paper: #fffdf7;
      --ink: #1e1a17;
      --muted: #6a6258;
      --accent: #b6522c;
      --accent-soft: #f1ddca;
      --line: #dccbb4;
      --chip: #efe4d3;
      --shadow: rgba(73, 45, 17, 0.12);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: "Noto Serif SC", "Songti SC", "STSong", serif;
      color: var(--ink);
      background:
        radial-gradient(circle at top left, rgba(182, 82, 44, 0.18), transparent 32%),
        radial-gradient(circle at top right, rgba(101, 134, 96, 0.14), transparent 26%),
        linear-gradient(180deg, #f3eadc 0%, var(--bg) 100%);
      min-height: 100vh;
    }
    .shell {
      max-width: 1280px;
      margin: 0 auto;
      padding: 32px 20px 64px;
    }
    .hero {
      display: grid;
      grid-template-columns: 1.2fr 0.8fr;
      gap: 18px;
      align-items: stretch;
      margin-bottom: 24px;
    }
    .panel {
      background: rgba(255, 253, 247, 0.82);
      border: 1px solid rgba(220, 203, 180, 0.86);
      border-radius: 24px;
      box-shadow: 0 22px 50px var(--shadow);
      backdrop-filter: blur(12px);
    }
    .hero-main {
      padding: 28px;
      position: relative;
      overflow: hidden;
    }
    .hero-main::after {
      content: "";
      position: absolute;
      inset: auto -32px -32px auto;
      width: 180px;
      height: 180px;
      background: radial-gradient(circle, rgba(182, 82, 44, 0.22), transparent 66%);
      pointer-events: none;
    }
    .eyebrow {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      font-size: 12px;
      letter-spacing: 0.18em;
      text-transform: uppercase;
      color: var(--accent);
      margin-bottom: 14px;
    }
    h1 {
      margin: 0 0 10px;
      font-size: clamp(32px, 5vw, 58px);
      line-height: 0.95;
      letter-spacing: -0.04em;
    }
    .hero-copy {
      max-width: 640px;
      margin: 0;
      font-size: 16px;
      line-height: 1.8;
      color: var(--muted);
    }
    .hero-side {
      padding: 24px;
      display: grid;
      gap: 14px;
    }
    .metric {
      padding: 16px 18px;
      border-radius: 18px;
      background: linear-gradient(180deg, rgba(255,255,255,0.76), rgba(241,221,202,0.88));
      border: 1px solid rgba(220, 203, 180, 0.72);
    }
    .metric label {
      display: block;
      color: var(--muted);
      font-size: 12px;
      margin-bottom: 8px;
      text-transform: uppercase;
      letter-spacing: 0.08em;
    }
    .metric strong {
      font-size: 28px;
      letter-spacing: -0.04em;
    }
    .toolbar {
      display: grid;
      grid-template-columns: minmax(0, 1.6fr) repeat(2, minmax(180px, 0.5fr));
      gap: 12px;
      margin-bottom: 20px;
    }
    .toolbar input, .toolbar select {
      width: 100%;
      appearance: none;
      border: 1px solid var(--line);
      background: rgba(255, 253, 247, 0.84);
      border-radius: 16px;
      padding: 14px 16px;
      font: inherit;
      color: var(--ink);
      outline: none;
    }
    .toolbar input:focus, .toolbar select:focus {
      border-color: var(--accent);
      box-shadow: 0 0 0 4px rgba(182, 82, 44, 0.12);
    }
    .list {
      display: grid;
      gap: 18px;
    }
    .card {
      position: relative;
      overflow: hidden;
      padding: 22px;
      border-radius: 24px;
      background: var(--paper);
      border: 1px solid rgba(220, 203, 180, 0.82);
      box-shadow: 0 18px 42px rgba(66, 47, 20, 0.08);
      transform: translateY(8px);
      opacity: 0;
      animation: rise 0.5s ease forwards;
    }
    .card::before {
      content: "";
      position: absolute;
      left: 0;
      top: 0;
      bottom: 0;
      width: 5px;
      background: linear-gradient(180deg, #d06c3f, #7f3b25);
    }
    .card-head {
      display: flex;
      justify-content: space-between;
      gap: 18px;
      align-items: flex-start;
      margin-bottom: 14px;
    }
    .card h2 {
      margin: 0;
      font-size: clamp(22px, 3vw, 32px);
      line-height: 1.1;
      letter-spacing: -0.03em;
    }
    .meta {
      color: var(--muted);
      font-size: 13px;
      line-height: 1.7;
      text-align: right;
      min-width: 180px;
    }
    .summary {
      margin: 0 0 16px;
      color: #433930;
      line-height: 1.75;
      font-size: 15px;
    }
    .chips {
      display: flex;
      gap: 8px;
      flex-wrap: wrap;
      margin: 0 0 16px;
    }
    .chip {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 8px 11px;
      border-radius: 999px;
      background: var(--chip);
      font-size: 12px;
      color: #5d4736;
      border: 1px solid rgba(214, 184, 151, 0.72);
    }
    .grid {
      display: grid;
      grid-template-columns: repeat(4, minmax(0, 1fr));
      gap: 10px;
      margin-bottom: 16px;
    }
    .score {
      border-radius: 18px;
      padding: 12px 14px;
      background: linear-gradient(180deg, rgba(241,221,202,0.62), rgba(255,255,255,0.84));
      border: 1px solid rgba(220, 203, 180, 0.7);
    }
    .score label {
      display: block;
      font-size: 12px;
      color: var(--muted);
      margin-bottom: 4px;
    }
    .score strong {
      font-size: 22px;
      letter-spacing: -0.04em;
    }
    .body {
      border-top: 1px dashed var(--line);
      padding-top: 14px;
      color: #2d2620;
      line-height: 1.82;
      white-space: pre-wrap;
      font-size: 14px;
      display: none;
    }
    .card.open .body { display: block; }
    .actions {
      display: flex;
      justify-content: space-between;
      align-items: center;
      gap: 12px;
      margin-top: 12px;
    }
    .toggle, .link {
      border: 0;
      background: none;
      color: var(--accent);
      font: inherit;
      cursor: pointer;
      padding: 0;
      text-decoration: none;
    }
    .empty {
      padding: 44px 20px;
      text-align: center;
      color: var(--muted);
      border: 1px dashed var(--line);
      border-radius: 24px;
      background: rgba(255, 253, 247, 0.6);
    }
    @keyframes rise {
      to {
        opacity: 1;
        transform: translateY(0);
      }
    }
    @media (max-width: 960px) {
      .hero { grid-template-columns: 1fr; }
      .toolbar { grid-template-columns: 1fr; }
      .card-head { flex-direction: column; }
      .meta { text-align: left; min-width: 0; }
      .grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
    }
    @media (max-width: 640px) {
      .shell { padding: 18px 14px 44px; }
      .hero-main, .hero-side, .card { padding: 18px; }
      .grid { grid-template-columns: 1fr 1fr; }
    }
  </style>
</head>
<body>
  <div class="shell">
    <section class="hero">
      <div class="panel hero-main">
        <div class="eyebrow">Cleaned Archive</div>
        <h1>只看已清洗的素材</h1>
        <p class="hero-copy">页面只展示已经完成 AI 清洗、分类、打分和打标签的内容。你可以直接按标题、类型和语言筛选，快速浏览可写素材。</p>
      </div>
      <div class="panel hero-side">
        <div class="metric">
          <label>当前展示</label>
          <strong id="metric-count">0</strong>
        </div>
        <div class="metric">
          <label>高价值候选</label>
          <strong id="metric-worth">0</strong>
        </div>
        <div class="metric">
          <label>默认来源</label>
          <strong>cleaned</strong>
        </div>
      </div>
    </section>

    <section class="toolbar">
      <input id="search" type="search" placeholder="搜索标题、摘要、标签、角度">
      <select id="type-filter">
        <option value="">全部类型</option>
      </select>
      <select id="lang-filter">
        <option value="">全部语言</option>
      </select>
    </section>

    <section id="list" class="list"></section>
  </div>

  <script>
    const state = {
      items: [],
      filtered: []
    };

    const searchInput = document.getElementById('search');
    const typeFilter = document.getElementById('type-filter');
    const langFilter = document.getElementById('lang-filter');
    const listNode = document.getElementById('list');
    const metricCount = document.getElementById('metric-count');
    const metricWorth = document.getElementById('metric-worth');

    function scoreTone(value) {
      if (value >= 8) return '高';
      if (value >= 6) return '中';
      return '低';
    }

    function formatTime(value) {
      if (!value) return '未知时间';
      const date = new Date(value);
      if (Number.isNaN(date.getTime())) return value;
      return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
      });
    }

    function uniqueValues(key) {
      return [...new Set(state.items.map(item => item[key]).filter(Boolean))].sort();
    }

    function renderFilters() {
      const types = uniqueValues('content_type');
      const langs = uniqueValues('language');

      typeFilter.innerHTML = '<option value="">全部类型</option>' + types.map(value => '<option value="' + value + '">' + value + '</option>').join('');
      langFilter.innerHTML = '<option value="">全部语言</option>' + langs.map(value => '<option value="' + value + '">' + value + '</option>').join('');
    }

    function filterItems() {
      const keyword = searchInput.value.trim().toLowerCase();
      const type = typeFilter.value;
      const lang = langFilter.value;

      state.filtered = state.items.filter(item => {
        const haystack = [
          item.cleaned_title,
          item.cleaned_summary,
          item.cleaned_content,
          item.angle_hint,
          item.ai_reason,
          ...(item.tags || []).map(tag => tag.name)
        ].join(' ').toLowerCase();

        if (keyword && !haystack.includes(keyword)) return false;
        if (type && item.content_type !== type) return false;
        if (lang && item.language !== lang) return false;
        return true;
      });

      renderList();
    }

    function toggleCard(id) {
      const node = document.getElementById('card-' + id);
      if (!node) return;
      node.classList.toggle('open');
    }

    function renderList() {
      metricCount.textContent = String(state.filtered.length);
      metricWorth.textContent = String(state.filtered.filter(item => item.writeworthy_score >= 7).length);

      if (!state.filtered.length) {
        listNode.innerHTML = '<div class="empty">当前没有符合条件的已清洗数据。</div>';
        return;
      }

      listNode.innerHTML = state.filtered.map((item, index) => {
        const tags = (item.tags || []).map(tag => '<span class="chip">' + tag.category + ' · ' + tag.name + '</span>').join('');
        const content = (item.cleaned_content || '').replace(/[&<>]/g, ch => ({'&':'&amp;','<':'&lt;','>':'&gt;'}[ch]));
        const summary = (item.cleaned_summary || '').replace(/[&<>]/g, ch => ({'&':'&amp;','<':'&lt;','>':'&gt;'}[ch]));
        const title = (item.cleaned_title || item.rss_title || '').replace(/[&<>]/g, ch => ({'&':'&amp;','<':'&lt;','>':'&gt;'}[ch]));
        const angle = (item.angle_hint || '未提供角度').replace(/[&<>]/g, ch => ({'&':'&amp;','<':'&lt;','>':'&gt;'}[ch]));
        const reason = (item.ai_reason || '未提供原因').replace(/[&<>]/g, ch => ({'&':'&amp;','<':'&lt;','>':'&gt;'}[ch]));

        return '<article class="card" id="card-' + item.id + '" style="animation-delay:' + (index * 40) + 'ms">' +
          '<div class="card-head">' +
            '<div><h2>' + title + '</h2></div>' +
            '<div class="meta">' +
              '<div>' + (item.rss_source_site || '未知来源') + '</div>' +
              '<div>' + formatTime(item.rss_published_at || item.updated_at) + '</div>' +
              '<div>' + (item.language || 'unknown') + ' / ' + (item.content_type || 'unknown') + '</div>' +
            '</div>' +
          '</div>' +
          '<p class="summary">' + summary + '</p>' +
          '<div class="chips">' +
            '<span class="chip">角度 · ' + angle + '</span>' +
            '<span class="chip">相关性 · ' + (item.is_relevant ? '是' : '否') + '</span>' +
            tags +
          '</div>' +
          '<div class="grid">' +
            '<div class="score"><label>重要性</label><strong>' + item.importance_score + '</strong></div>' +
            '<div class="score"><label>可写性</label><strong>' + item.writeworthy_score + '</strong></div>' +
            '<div class="score"><label>质量</label><strong>' + item.quality_score + '</strong></div>' +
            '<div class="score"><label>评级</label><strong>' + scoreTone(item.writeworthy_score) + '</strong></div>' +
          '</div>' +
          '<div class="chips"><span class="chip">判断 · ' + reason + '</span></div>' +
          '<div class="actions">' +
            '<button class="toggle" onclick="toggleCard(' + item.id + ')">展开 / 收起正文</button>' +
            '<a class="link" href="' + (item.canonical_url || item.final_url || '#') + '" target="_blank" rel="noreferrer">打开原文</a>' +
          '</div>' +
          '<div class="body">' + content + '</div>' +
        '</article>';
      }).join('');
    }

    async function load() {
      const response = await fetch('/api/cleaned?limit=200');
      if (!response.ok) {
        listNode.innerHTML = '<div class="empty">加载失败：' + response.status + '</div>';
        return;
      }

      const payload = await response.json();
      state.items = payload.items || [];
      renderFilters();
      filterItems();
    }

    searchInput.addEventListener('input', filterItems);
    typeFilter.addEventListener('change', filterItems);
    langFilter.addEventListener('change', filterItems);

    load().catch(error => {
      listNode.innerHTML = '<div class="empty">加载失败：' + error.message + '</div>';
    });
  </script>
</body>
</html>`))

func NewServer(db *sql.DB) *Server {
	return &Server{
		repo:      content.NewRepository(db),
		queryRepo: query.NewRepository(db),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/admin", s.handleAdminIndex)
	mux.HandleFunc("/api/cleaned", s.handleCleaned)
	mux.HandleFunc("GET /api/admin/stats", s.handleAdminStats)
	mux.HandleFunc("POST /api/admin/reset-failed", s.handleAdminResetFailed)
	mux.HandleFunc("GET /api/admin/queries", s.handleAdminListQueries)
	mux.HandleFunc("POST /api/admin/queries", s.handleAdminCreateQuery)
	mux.HandleFunc("PATCH /api/admin/queries/{id}", s.handleAdminUpdateQuery)
	mux.HandleFunc("DELETE /api/admin/queries/{id}", s.handleAdminDeleteQuery)
	mux.HandleFunc("/healthz", s.handleHealthz)
	return mux
}

func (s *Server) Serve(ctx context.Context, addr string) error {
	server := &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	err := server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) && ctx.Err() != nil {
		return nil
	}
	return err
}

func (s *Server) handleIndex(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pageTemplate.Execute(w, nil); err != nil {
		http.Error(w, "渲染页面失败", http.StatusInternalServerError)
	}
}

func (s *Server) handleCleaned(w http.ResponseWriter, r *http.Request) {
	limit := 200
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			http.Error(w, "limit 参数不合法", http.StatusBadRequest)
			return
		}
		if parsed > 500 {
			parsed = 500
		}
		limit = parsed
	}

	items, err := s.repo.ListCleaned(r.Context(), limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("读取 cleaned 数据失败: %v", err), http.StatusInternalServerError)
		return
	}

	response := make([]cleanedItemResponse, 0, len(items))
	for _, item := range items {
		response = append(response, cleanedItemResponse{
			ID:               item.ID,
			QueryText:        item.QueryText,
			RSSTitle:         item.RSSTitle,
			RSSSourceSite:    item.RSSSourceSite,
			RSSPublishedAt:   formatNullableTime(item.RSSPublishedAt),
			FinalURL:         item.FinalURL,
			CanonicalURL:     item.CanonicalURL,
			CleanedTitle:     item.CleanedTitle,
			CleanedSummary:   item.CleanedSummary,
			CleanedContent:   item.CleanedContent,
			Language:         item.Language,
			ContentType:      item.ContentType,
			QualityScore:     item.QualityScore,
			ImportanceScore:  item.ImportanceScore,
			WriteworthyScore: item.WriteworthyScore,
			IsRelevant:       item.IsRelevant,
			AngleHint:        item.AngleHint,
			AIReason:         item.AIReason,
			UpdatedAt:        item.UpdatedAt.Format(time.RFC3339),
			Tags:             item.Tags,
		})
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(cleanedListResponse{
		Source: "cleaned",
		Limit:  limit,
		Count:  len(response),
		Items:  response,
	}); err != nil {
		http.Error(w, "编码 cleaned 数据失败", http.StatusInternalServerError)
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func formatNullableTime(value sql.NullTime) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format(time.RFC3339)
}

// ── Admin ─────────────────────────────────────────────────────────────────────

var adminTemplate = template.Must(template.New("admin").Parse(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Auto Search 后台管理</title>
  <style>
    :root {
      --bg: #f6f2e8; --paper: #fffdf7; --ink: #1e1a17; --muted: #6a6258;
      --accent: #b6522c; --accent-soft: #f1ddca; --line: #dccbb4;
      --chip: #efe4d3; --shadow: rgba(73,45,17,.12); --danger: #c0392b;
    }
    * { box-sizing: border-box; }
    body { margin: 0; font-family: system-ui, -apple-system, sans-serif; color: var(--ink); background: linear-gradient(180deg, #f3eadc 0%, var(--bg) 100%); min-height: 100vh; }
    .shell { max-width: 1200px; margin: 0 auto; padding: 0 20px 64px; }
    nav { display: flex; align-items: center; gap: 24px; padding: 18px 0; border-bottom: 1px solid var(--line); margin-bottom: 28px; }
    .nav-brand { font-size: 18px; font-weight: 700; letter-spacing: -.02em; }
    nav a { color: var(--muted); text-decoration: none; font-size: 14px; }
    nav a.active { color: var(--accent); font-weight: 600; }
    nav a:hover { color: var(--ink); }
    h2 { margin: 0; font-size: 20px; letter-spacing: -.02em; }
    .stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 14px; margin-bottom: 32px; }
    .stat { background: rgba(255,253,247,.9); border: 1px solid rgba(220,203,180,.8); border-radius: 18px; padding: 18px 20px; box-shadow: 0 8px 24px var(--shadow); }
    .stat label { display: block; font-size: 12px; color: var(--muted); text-transform: uppercase; letter-spacing: .08em; margin-bottom: 8px; }
    .stat strong { font-size: 32px; letter-spacing: -.04em; }
    .section { background: rgba(255,253,247,.9); border: 1px solid rgba(220,203,180,.8); border-radius: 20px; padding: 24px; box-shadow: 0 8px 24px var(--shadow); }
    .section-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
    .btn { border: 1px solid var(--line); background: var(--paper); color: var(--ink); border-radius: 10px; padding: 8px 16px; font: inherit; font-size: 13px; cursor: pointer; }
    .btn:hover { background: var(--accent-soft); border-color: var(--accent); }
    .btn-primary { background: var(--accent); color: #fff; border-color: var(--accent); }
    .btn-primary:hover { opacity: .88; }
    .btn-danger { color: var(--danger); border-color: #f5c0bb; }
    .btn-danger:hover { background: #fdf0ef; border-color: var(--danger); }
    .btn-sm { padding: 5px 11px; font-size: 12px; border-radius: 8px; }
    table { width: 100%; border-collapse: collapse; font-size: 14px; }
    th { text-align: left; padding: 10px 12px; font-size: 11px; text-transform: uppercase; letter-spacing: .08em; color: var(--muted); border-bottom: 1px solid var(--line); }
    td { padding: 12px; border-bottom: 1px solid rgba(220,203,180,.5); vertical-align: middle; }
    tr:last-child td { border-bottom: none; }
    tr:hover td { background: rgba(241,221,202,.25); }
    .badge { display: inline-block; padding: 3px 9px; border-radius: 999px; font-size: 11px; font-weight: 600; }
    .badge-on { background: #d4edda; color: #155724; }
    .badge-off { background: #f8d7da; color: #721c24; }
    .form-row { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 12px; margin-bottom: 14px; }
    .form-row.full { grid-template-columns: 1fr; }
    .field { display: flex; flex-direction: column; gap: 4px; }
    .field label { font-size: 12px; color: var(--muted); }
    .field input, .field select { border: 1px solid var(--line); background: var(--paper); border-radius: 10px; padding: 9px 12px; font: inherit; font-size: 13px; outline: none; }
    .field input:focus, .field select:focus { border-color: var(--accent); box-shadow: 0 0 0 3px rgba(182,82,44,.1); }
    .inline-form { background: rgba(241,221,202,.3); border: 1px dashed var(--line); border-radius: 14px; padding: 18px; margin-bottom: 18px; }
    .inline-form .actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 12px; }
    .modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,.35); display: flex; align-items: center; justify-content: center; z-index: 100; }
    .modal-box { background: var(--paper); border-radius: 20px; padding: 28px; width: 100%; max-width: 560px; box-shadow: 0 32px 80px rgba(0,0,0,.25); }
    .modal-box h3 { margin: 0 0 20px; font-size: 18px; }
    .modal-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 18px; }
    .hidden { display: none !important; }
    @media (max-width: 640px) {
      .stats { grid-template-columns: 1fr 1fr; }
      table { display: block; overflow-x: auto; }
    }
  </style>
</head>
<body>
  <div class="shell">
    <nav>
      <span class="nav-brand">Auto Search</span>
      <a href="/">素材库</a>
      <a href="/admin" class="active">后台管理</a>
    </nav>

    <section class="stats" id="stats">
      <div class="stat"><label>待抓取</label><strong id="s-pending">-</strong></div>
      <div class="stat"><label>已抓取</label><strong id="s-extracted">-</strong></div>
      <div class="stat" style="position:relative">
        <label>抓取失败</label><strong id="s-extract-failed">-</strong>
        <button class="btn btn-sm" style="position:absolute;top:14px;right:14px;font-size:11px" onclick="resetFailed()">重置</button>
      </div>
      <div class="stat"><label>已清洗</label><strong id="s-cleaned">-</strong></div>
      <div class="stat"><label>总数</label><strong id="s-total">-</strong></div>
    </section>

    <section class="section">
      <div class="section-head">
        <h2>搜索词管理</h2>
        <button class="btn btn-primary btn-sm" onclick="openAdd()">+ 新增</button>
      </div>

      <div id="add-form" class="inline-form hidden">
        <div class="form-row">
          <div class="field"><label>名称（唯一）</label><input id="f-name" placeholder="openai" /></div>
          <div class="field"><label>搜索词</label><input id="f-query" placeholder="OpenAI when:1d" /></div>
          <div class="field"><label>Lang</label>
            <select id="f-lang"><option value="en">en</option><option value="zh">zh</option><option value="ja">ja</option></select>
          </div>
          <div class="field"><label>Region</label>
            <select id="f-region"><option value="US">US</option><option value="CN">CN</option><option value="JP">JP</option></select>
          </div>
          <div class="field"><label>优先级</label><input id="f-priority" type="number" value="0" /></div>
          <div class="field"><label>状态</label>
            <select id="f-enabled"><option value="true">启用</option><option value="false">禁用</option></select>
          </div>
        </div>
        <div class="inline-form actions">
          <button class="btn btn-sm" onclick="closeAdd()">取消</button>
          <button class="btn btn-primary btn-sm" onclick="createQuery()">保存</button>
        </div>
      </div>

      <table>
        <thead>
          <tr>
            <th>ID</th><th>名称</th><th>搜索词</th><th>Lang</th><th>Region</th>
            <th>优先级</th><th>状态</th><th>操作</th>
          </tr>
        </thead>
        <tbody id="queries-body"></tbody>
      </table>
    </section>
  </div>

  <div id="modal" class="modal-overlay hidden">
    <div class="modal-box">
      <h3>编辑搜索词</h3>
      <input type="hidden" id="e-id" />
      <div class="form-row">
        <div class="field"><label>名称</label><input id="e-name" /></div>
        <div class="field"><label>搜索词</label><input id="e-query" /></div>
        <div class="field"><label>Lang</label>
          <select id="e-lang"><option value="en">en</option><option value="zh">zh</option><option value="ja">ja</option></select>
        </div>
        <div class="field"><label>Region</label>
          <select id="e-region"><option value="US">US</option><option value="CN">CN</option><option value="JP">JP</option></select>
        </div>
        <div class="field"><label>优先级</label><input id="e-priority" type="number" /></div>
        <div class="field"><label>状态</label>
          <select id="e-enabled"><option value="true">启用</option><option value="false">禁用</option></select>
        </div>
      </div>
      <div class="modal-actions">
        <button class="btn btn-sm" onclick="closeModal()">取消</button>
        <button class="btn btn-primary btn-sm" onclick="updateQuery()">保存</button>
      </div>
    </div>
  </div>

  <script>
    async function loadStats() {
      const res = await fetch('/api/admin/stats');
      if (!res.ok) return;
      const data = await res.json();
      const map = {};
      let total = 0;
      for (const item of (data.counts || [])) {
        map[item.status] = item.count;
        total += item.count;
      }
      document.getElementById('s-pending').textContent = map['pending'] || 0;
      document.getElementById('s-extracted').textContent = map['extracted'] || 0;
      document.getElementById('s-extract-failed').textContent = map['extract_failed'] || 0;
      document.getElementById('s-cleaned').textContent = map['cleaned'] || 0;
      document.getElementById('s-total').textContent = total;
    }

    async function loadQueries() {
      const res = await fetch('/api/admin/queries');
      if (!res.ok) return;
      const data = await res.json();
      const tbody = document.getElementById('queries-body');
      tbody.innerHTML = (data.items || []).map(q => {
        const badge = q.enabled
          ? '<span class="badge badge-on">启用</span>'
          : '<span class="badge badge-off">禁用</span>';
        const esc = s => String(s || '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
        return '<tr>' +
          '<td>' + q.id + '</td>' +
          '<td><strong>' + esc(q.name) + '</strong></td>' +
          '<td style="max-width:260px;word-break:break-all">' + esc(q.query_text) + '</td>' +
          '<td>' + esc(q.lang) + '</td>' +
          '<td>' + esc(q.region) + '</td>' +
          '<td>' + q.priority + '</td>' +
          '<td>' + badge + '</td>' +
          '<td style="white-space:nowrap">' +
            '<button class="btn btn-sm" style="margin-right:6px" onclick="openEdit(' + JSON.stringify(q).replace(/</g,'\\u003c') + ')">编辑</button>' +
            '<button class="btn btn-sm btn-danger" onclick="deleteQuery(' + q.id + ')">删除</button>' +
          '</td>' +
        '</tr>';
      }).join('');
    }

    function openAdd() { document.getElementById('add-form').classList.remove('hidden'); }
    function closeAdd() { document.getElementById('add-form').classList.add('hidden'); }

    async function createQuery() {
      const body = {
        name: document.getElementById('f-name').value.trim(),
        query_text: document.getElementById('f-query').value.trim(),
        lang: document.getElementById('f-lang').value,
        region: document.getElementById('f-region').value,
        priority: parseInt(document.getElementById('f-priority').value, 10) || 0,
        enabled: document.getElementById('f-enabled').value === 'true',
      };
      if (!body.name || !body.query_text) { alert('名称和搜索词不能为空'); return; }
      const res = await fetch('/api/admin/queries', { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify(body) });
      if (!res.ok) { alert('创建失败: ' + await res.text()); return; }
      closeAdd();
      loadQueries();
    }

    function openEdit(q) {
      document.getElementById('e-id').value = q.id;
      document.getElementById('e-name').value = q.name;
      document.getElementById('e-query').value = q.query_text;
      document.getElementById('e-lang').value = q.lang;
      document.getElementById('e-region').value = q.region;
      document.getElementById('e-priority').value = q.priority;
      document.getElementById('e-enabled').value = q.enabled ? 'true' : 'false';
      document.getElementById('modal').classList.remove('hidden');
    }
    function closeModal() { document.getElementById('modal').classList.add('hidden'); }

    async function updateQuery() {
      const id = document.getElementById('e-id').value;
      const body = {
        name: document.getElementById('e-name').value.trim(),
        query_text: document.getElementById('e-query').value.trim(),
        lang: document.getElementById('e-lang').value,
        region: document.getElementById('e-region').value,
        priority: parseInt(document.getElementById('e-priority').value, 10) || 0,
        enabled: document.getElementById('e-enabled').value === 'true',
      };
      const res = await fetch('/api/admin/queries/' + id, { method: 'PATCH', headers: {'Content-Type':'application/json'}, body: JSON.stringify(body) });
      if (!res.ok) { alert('更新失败: ' + await res.text()); return; }
      closeModal();
      loadQueries();
    }

    async function deleteQuery(id) {
      if (!confirm('确定删除该搜索词？')) return;
      const res = await fetch('/api/admin/queries/' + id, { method: 'DELETE' });
      if (!res.ok) { alert('删除失败: ' + await res.text()); return; }
      loadQueries();
    }

    async function resetFailed() {
      const count = parseInt(document.getElementById('s-extract-failed').textContent, 10) || 0;
      if (!confirm('将 ' + count + ' 条抓取失败记录重置为待抓取？')) return;
      const res = await fetch('/api/admin/reset-failed', { method: 'POST' });
      if (!res.ok) { alert('重置失败: ' + await res.text()); return; }
      const data = await res.json();
      alert('已重置 ' + data.reset + ' 条记录');
      loadStats();
    }

    loadStats();
    loadQueries();
  </script>
</body>
</html>`))

func (s *Server) handleAdminIndex(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := adminTemplate.Execute(w, nil); err != nil {
		http.Error(w, "渲染后台页面失败", http.StatusInternalServerError)
	}
}

func (s *Server) handleAdminResetFailed(w http.ResponseWriter, r *http.Request) {
	n, err := s.repo.ResetFailedToPending(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("重置失败: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"reset": n})
}

func (s *Server) handleAdminStats(w http.ResponseWriter, r *http.Request) {
	counts, err := s.repo.CountByStatus(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("统计状态失败: %v", err), http.StatusInternalServerError)
		return
	}

	type countItem struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}
	items := make([]countItem, 0, len(counts))
	for _, c := range counts {
		items = append(items, countItem{Status: c.Status, Count: c.Count})
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"counts": items})
}

type queryResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	QueryText string `json:"query_text"`
	Lang      string `json:"lang"`
	Region    string `json:"region"`
	Enabled   bool   `json:"enabled"`
	Priority  int    `json:"priority"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type queryRequest struct {
	Name      string `json:"name"`
	QueryText string `json:"query_text"`
	Lang      string `json:"lang"`
	Region    string `json:"region"`
	Enabled   bool   `json:"enabled"`
	Priority  int    `json:"priority"`
}

func (s *Server) handleAdminListQueries(w http.ResponseWriter, r *http.Request) {
	items, err := s.queryRepo.ListAll(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("查询 queries 失败: %v", err), http.StatusInternalServerError)
		return
	}

	resp := make([]queryResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, queryResponse{
			ID:        item.ID,
			Name:      item.Name,
			QueryText: item.QueryText,
			Lang:      item.Lang,
			Region:    item.Region,
			Enabled:   item.Enabled,
			Priority:  item.Priority,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"items": resp})
}

func (s *Server) handleAdminCreateQuery(w http.ResponseWriter, r *http.Request) {
	var req queryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "请求体解析失败", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.QueryText == "" {
		http.Error(w, "name 和 query_text 不能为空", http.StatusBadRequest)
		return
	}
	if req.Lang == "" {
		req.Lang = "en"
	}
	if req.Region == "" {
		req.Region = "US"
	}

	id, err := s.queryRepo.Insert(r.Context(), query.FeedQuery{
		Name:      req.Name,
		QueryText: req.QueryText,
		Lang:      req.Lang,
		Region:    req.Region,
		Enabled:   req.Enabled,
		Priority:  req.Priority,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("创建 query 失败: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"id": id})
}

func (s *Server) handleAdminUpdateQuery(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "id 不合法", http.StatusBadRequest)
		return
	}

	var req queryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "请求体解析失败", http.StatusBadRequest)
		return
	}

	if err := s.queryRepo.Update(r.Context(), query.FeedQuery{
		ID:        id,
		Name:      req.Name,
		QueryText: req.QueryText,
		Lang:      req.Lang,
		Region:    req.Region,
		Enabled:   req.Enabled,
		Priority:  req.Priority,
	}); err != nil {
		http.Error(w, fmt.Sprintf("更新 query 失败: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminDeleteQuery(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "id 不合法", http.StatusBadRequest)
		return
	}

	if err := s.queryRepo.Delete(r.Context(), id); err != nil {
		http.Error(w, fmt.Sprintf("删除 query 失败: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
