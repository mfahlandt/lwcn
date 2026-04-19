# Sponsored & Partner Content Guide

This document describes how to add **clearly labeled** sponsored or partner
placements to a weekly LWCN newsletter edition without compromising the
neutrality of the editorial sections.

---

## TL;DR

1. **Never edit the AI-generated editorial sections** to insert sponsor copy.
2. Add sponsor placements with the `{{< sponsored >}}` Hugo shortcode in a
   dedicated section at the **end** of the newsletter Markdown file.
3. The shortcode renders a visually distinct, legally compliant block
   (`rel="sponsored nofollow"`, ARIA-labeled, linked to the sponsorship
   policy).
4. The AI processor is instructed **never** to generate sponsor content,
   so it is safe to run the pipeline as usual.

---

## Legal background (Germany)

- **§ 22 MStV** — advertising in journalistic-editorial telemedia must be
  clearly separated from editorial content and labeled.
- **§ 6 DDG** (Digitale-Dienste-Gesetz, successor to § 6 TMG) — commercial
  communication must be clearly identifiable.
- **Google SEO** — paid links must carry `rel="sponsored"` (optionally also
  `nofollow`).

All of this is handled by the shortcode — you just have to use it.

---

## The `{{< sponsored >}}` shortcode

Location: `website/layouts/shortcodes/sponsored.html`
Styles:   `.sponsored` block in `website/static/css/style.css`

### Parameters

| Parameter  | Required | Default       | Description                                          |
|------------|----------|---------------|------------------------------------------------------|
| `sponsor`  | optional | `""`          | Sponsor name shown next to the badge                 |
| `url`      | optional | `""`          | If set, the sponsor name becomes a link (rel="sponsored nofollow") |
| `kind`     | optional | `"sponsored"` | `"sponsored"` or `"partner"` — changes the badge     |

### Minimal example

```markdown
{{< sponsored sponsor="Acme Corp" url="https://acme.example/" >}}
Acme Corp releases **KubeBrush v1.0** — GA release of their Kubernetes
operator for drift detection. Free tier available for CNCF projects.
{{< /sponsored >}}
```

### Partner variant

```markdown
{{< sponsored kind="partner" sponsor="KubeCon EU 2026" url="https://kubecon.io/" >}}
**KubeCon + CloudNativeCon Europe 2026** — Call for Papers open until
April 30. LWCN readers can use code `LWCN10` for a 10% registration
discount.
{{< /sponsored >}}
```

### Anonymous / house-ad variant

Leave `sponsor` blank to render a generic labeled block (useful for
cross-promoting your own community events, conferences you attend, etc.):

```markdown
{{< sponsored >}}
I am speaking at the Cloud Native Rejekts EU on May 18 in London.
Come by if you want to chat about LWCN or the Cloud Native ecosystem.
{{< /sponsored >}}
```

---

## Where to put the block inside a newsletter file

Given a file like `website/content/newsletter/2026-week-17.md`:

```markdown
---
title: "Week 17 - April 2026"
# ... existing AI-generated frontmatter ...
---

## 👋 Welcome
... AI-generated ...

## 🚀 Notable Releases
... AI-generated ...

## 📰 This Week in Cloud Native
... AI-generated ...

## 💬 Community Buzz
... AI-generated ...

## 📊 Numbers of the Week
... AI-generated ...

<!-- ==================== SPONSORED SECTION ==================== -->
<!-- Manually inserted post-generation. Do NOT edit editorial sections above. -->

## 💼 Sponsored

{{< sponsored sponsor="Acme Corp" url="https://acme.example/" >}}
One-sentence hook. Concrete fact. Link.
{{< /sponsored >}}

<!-- The "View all articles" link below is auto-added by the pipeline. -->
📚 **[View all articles from this week →](https://lwcn.dev/newsletter/2026-week-17/articles/)**
```

**Rules of thumb:**

- Place the `## 💼 Sponsored` heading **after** `## 📊 Numbers of the Week`
  and **before** the articles link.
- Keep the sponsored block copy **factual and short** — 1-3 sentences plus
  a link. Do not let sponsors write marketing paragraphs; rewrite them to
  the same neutral, fact-first tone as editorial content.
- Maximum **one** sponsored block per edition.

---

## What the pipeline will / will not do

### The AI processor (`ai-processor`) will NEVER:

- Insert `{{< sponsored >}}` shortcodes
- Add `[Sponsored]`, `[Partner]`, `Ad:`, `Promoted:` or
  "Brought to you by" tags
- Mention sponsors by name unless the sponsor is already mentioned in
  editorial source data

This is enforced in `internal/ai/gemini.go` → rule #10 of the newsletter
prompt. If you ever see the AI emit a sponsor-looking block, that is a
bug — file an issue.

### The social publisher (`social-publisher`) will NEVER:

- Include sponsor copy in the tweet / Bluesky / LinkedIn-short posts.
  The short posts are derived from the long LinkedIn article, which the AI
  generates **before** you add the sponsor block, so sponsors never leak
  into social teasers by accident.

If you want to promote the sponsor on social media, do that as a **separate**
post from your own account, not via the LWCN pipeline.

---

## Disclosure checklist (per sponsored edition)

Before merging a PR that contains a sponsored block, confirm:

- [ ] The block uses the `{{< sponsored >}}` shortcode (not plain HTML, not
      inline into editorial text).
- [ ] The `sponsor` parameter is set (unless it is a house ad).
- [ ] The outbound link was NOT manually stripped of `rel="sponsored nofollow"`.
- [ ] The copy is factual, contains no hype words, and would pass the
      same editorial neutrality rules as editorial content.
- [ ] Any material relationship between the sponsor and the publisher is
      disclosed in the block itself (e.g. "Acme is a client of the
      publisher").
- [ ] The newsletter still reads naturally without the sponsor block
      (i.e. the sponsor is additive, not a replacement for editorial
      coverage).

---

## Pricing, sales, contracts

Out of scope for this repo. Track deals privately. What lives in this
repo is purely the **publishing mechanism** and the **transparency
policy**.

## Links

- Sponsorship policy (public): <https://lwcn.dev/about/#independence--sponsorship>
- Imprint (advertising disclosure): <https://lwcn.dev/impressum/#advertising--sponsored-content>
- Shortcode source: [`website/layouts/shortcodes/sponsored.html`](../website/layouts/shortcodes/sponsored.html)
- CSS: [`website/static/css/style.css`](../website/static/css/style.css) (`.sponsored` block)
- AI prompt rule: [`internal/ai/gemini.go`](../internal/ai/gemini.go) → `buildPrompt` rule #10

