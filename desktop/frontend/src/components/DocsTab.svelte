<!-- desktop/frontend/src/components/DocsTab.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'
  import { marked } from 'marked'
  import indexJson from '../docs/index.json'

  interface Article {
    id: string
    title: string
    file: string
  }

  interface Category {
    id: string
    title: string
    articles: Article[]
  }

  let categories: Category[] = []
  let activeArticle: Article | null = null
  let activeCategory: Category | null = null
  let renderedHtml = ''
  let loading = false

  // Import all markdown files eagerly so Vite bundles them
  const mdModules = import.meta.glob('../docs/**/*.md', { as: 'raw', eager: true }) as Record<string, string>

  onMount(() => {
    categories = indexJson.categories
    if (categories.length > 0 && categories[0].articles.length > 0) {
      selectArticle(categories[0], categories[0].articles[0])
    }
  })

  async function selectArticle(category: Category, article: Article) {
    activeCategory = category
    activeArticle = article
    loading = true
    try {
      const raw = mdModules[`../docs/${article.file}`] ?? '# Not found\n\nThis article could not be loaded.'
      const dirty = await marked.parse(raw)
      // Strip scripts from rendered HTML
      renderedHtml = dirty.replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '')
    } finally {
      loading = false
    }
  }
</script>

<div class="docs-tab">
  <div class="tab-hero">
    <div class="hero-title">{activeArticle?.title ?? 'Documentation'}</div>
    <div class="hero-subtitle">
      {#if activeCategory && activeArticle}
        {activeCategory.title} › {activeArticle.title}
      {:else}
        Offline documentation
      {/if}
    </div>
  </div>

  <div class="docs-body">
    <nav class="docs-nav">
      {#each categories as category}
        <div class="nav-category">
          <div class="category-label">{category.title}</div>
          {#each category.articles as article}
            <button
              class="article-link"
              class:active={activeArticle?.id === article.id}
              on:click={() => selectArticle(category, article)}
            >
              {article.title}
            </button>
          {/each}
        </div>
      {/each}
    </nav>

    <div class="docs-content">
      {#if loading}
        <div class="loading">Loading…</div>
      {:else}
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        {@html renderedHtml}
      {/if}
    </div>
  </div>
</div>

<style>
  .docs-tab {
    display: flex;
    flex-direction: column;
    flex: 1;
    overflow: hidden;
  }

  .tab-hero {
    background: linear-gradient(135deg, var(--bg-hero-start), var(--bg-hero-end));
    padding: 18px 22px 16px;
    border-bottom: 1px solid var(--border-hero);
    flex-shrink: 0;
  }
  .hero-title {
    font-size: 15px;
    font-weight: 800;
    letter-spacing: -0.3px;
    color: var(--text-primary);
    margin-bottom: 4px;
  }
  .hero-subtitle {
    font-size: 11px;
    font-weight: 500;
    color: var(--accent);
  }

  .docs-body {
    display: flex;
    flex: 1;
    overflow: hidden;
  }

  .docs-nav {
    width: 160px;
    flex-shrink: 0;
    border-right: 1px solid var(--border);
    overflow-y: auto;
    padding: 12px 8px;
    display: flex;
    flex-direction: column;
    gap: 16px;
    background: var(--bg-content);
  }

  .nav-category { display: flex; flex-direction: column; gap: 2px; }

  .category-label {
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.6px;
    color: var(--text-muted);
    padding: 0 8px;
    margin-bottom: 4px;
  }

  .article-link {
    display: block;
    width: 100%;
    text-align: left;
    padding: 6px 8px;
    border: none;
    border-radius: 6px;
    background: transparent;
    font-size: 12px;
    color: var(--text-secondary);
    cursor: pointer;
    transition: background 0.1s, color 0.1s;
  }
  .article-link:hover { background: var(--accent-bg); color: var(--accent); }
  .article-link.active { background: var(--accent-bg); color: var(--accent); font-weight: 600; }

  .docs-content {
    flex: 1;
    overflow-y: auto;
    padding: 24px 28px;
    background: var(--bg-content);
    color: var(--text-primary);
    font-size: 13px;
    line-height: 1.7;
  }

  .loading { color: var(--text-muted); font-size: 12px; }

  /* Markdown content styles */
  .docs-content :global(h1) { font-size: 18px; font-weight: 800; margin-bottom: 16px; color: var(--text-primary); }
  .docs-content :global(h2) { font-size: 14px; font-weight: 700; margin: 20px 0 10px; color: var(--text-primary); }
  .docs-content :global(h3) { font-size: 13px; font-weight: 600; margin: 16px 0 8px; color: var(--text-primary); }
  .docs-content :global(p) { margin-bottom: 12px; }
  .docs-content :global(ul), .docs-content :global(ol) { padding-left: 20px; margin-bottom: 12px; }
  .docs-content :global(li) { margin-bottom: 4px; }
  .docs-content :global(code) { background: var(--bg-row); padding: 2px 5px; border-radius: 4px; font-size: 11px; font-family: monospace; }
  .docs-content :global(pre) { background: var(--bg-row); border: 1px solid var(--border); border-radius: 8px; padding: 12px; overflow-x: auto; margin-bottom: 12px; }
  .docs-content :global(pre code) { background: none; padding: 0; }
  .docs-content :global(hr) { border: none; border-top: 1px solid var(--border); margin: 20px 0; }
  .docs-content :global(a) { color: var(--accent); text-decoration: none; }
  .docs-content :global(a:hover) { text-decoration: underline; }
  .docs-content :global(strong) { font-weight: 700; }
</style>
