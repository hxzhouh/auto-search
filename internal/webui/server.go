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
)

type Server struct {
	repo *content.Repository
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
	return &Server{repo: content.NewRepository(db)}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/cleaned", s.handleCleaned)
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
