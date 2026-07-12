// bl — 多平台发布辅助脚本
// node publish-helper.js <platform>
// 平台: hn | reddit-golang | reddit-cool | reddit-cli | v2ex | twitter

const { chromium } = require('playwright');
const fs = require('fs');
const path = require('path');

const DATA_DIR = path.join(__dirname, 'playwright-data');
const CONTENT_DIR = path.join(__dirname, 'promotion');

const URLS = {
  'hn': 'https://news.ycombinator.com/submit',
  'reddit-golang': 'https://old.reddit.com/r/golang/submit',
  'reddit-cool': 'https://old.reddit.com/r/coolgithubprojects/submit',
  'reddit-cli': 'https://old.reddit.com/r/commandline/submit',
  'v2ex': 'https://v2ex.com/go/create',
  'twitter': 'https://twitter.com/compose/tweet',
};

const PLATFORMS = {
  'hn': {
    name: 'Hacker News',
    check: async (page) => {
      const t = await page.evaluate(() => document.body.innerText.substring(0,50));
      if (t.includes('Sorry')) {
        console.log('\n⚠️  需要登录 HN (页面上方 login)');
        console.log('   等待中...');
        await page.waitForFunction(() => document.querySelector('input[name="title"]'), { timeout: 300000 });
      }
    },
    fill: async (page) => {
      const content = fs.readFileSync(path.join(CONTENT_DIR, 'hackernews_showhn.md'), 'utf-8');
      const m = content.match(/## Plain text version[\s\S]*?(?=##|---|$)/);
      const body = m ? m[0].replace(/## Plain text version.*\n/, '').trim() : '';
      await page.fill('input[name="title"]', 'Show HN: bl – terminal dictionary with LLM, grammar parsing, offline dicts');
      await page.fill('input[name="url"]', 'https://github.com/xieguaiwu/bl');
      await page.fill('textarea[name="text"]', body);
      console.log('✅ HN 已填好');
    }
  },
  'reddit-golang': {
    name: 'Reddit r/golang',
    check: async (page) => {
      if (page.url().includes('login')) {
        console.log('\n⚠️  需要登录 Reddit');
        await page.waitForURL('**/submit', { timeout: 300000 });
      }
    },
    fill: async (page) => {
      const content = fs.readFileSync(path.join(CONTENT_DIR, 'reddit.md'), 'utf-8');
      const sec = content.split(/---+/).find(s => s.includes('r/golang')) || content;
      await page.fill('input[name="title"]', 'bl — a Go terminal dictionary with goquery, pure-Go SQLite, and LLM integration');
      await page.fill('input[name="url"]', 'https://github.com/xieguaiwu/bl');
      const ta = await page.$('textarea[name="text"]');
      if (ta) await ta.fill(sec);
      console.log('✅ r/golang 已填好');
    }
  },
  'reddit-cool': {
    name: 'Reddit r/coolgithubprojects',
    check: async (page) => {
      if (page.url().includes('login')) {
        console.log('\n⚠️  需要登录 Reddit');
        await page.waitForURL('**/submit', { timeout: 300000 });
      }
    },
    fill: async (page) => {
      await page.fill('input[name="title"]', 'bl — terminal dictionary with LLM translation, grammar parsing, and offline support');
      await page.fill('input[name="url"]', 'https://github.com/xieguaiwu/bl');
      console.log('✅ r/coolgithubprojects 已填好 (link post)');
    }
  },
  'reddit-cli': {
    name: 'Reddit r/commandline',
    check: async (page) => {
      if (page.url().includes('login')) {
        console.log('\n⚠️  需要登录 Reddit');
        await page.waitForURL('**/submit', { timeout: 300000 });
      }
    },
    fill: async (page) => {
      const content = fs.readFileSync(path.join(CONTENT_DIR, 'reddit.md'), 'utf-8');
      const sec = content.split(/---+/).find(s => s.includes('commandline')) || content;
      await page.fill('input[name="title"]', 'bl — terminal dictionary with offline support, LLM translation, and grammar analysis');
      await page.fill('input[name="url"]', 'https://github.com/xieguaiwu/bl');
      const ta = await page.$('textarea[name="text"]');
      if (ta) await ta.fill(sec);
      console.log('✅ r/commandline 已填好');
    }
  },
  'v2ex': {
    name: 'V2EX 分享创造',
    check: async (page) => {
      if (page.url().includes('signin')) {
        console.log('\n⚠️  需要登录 V2EX');
        await page.waitForURL('**/create', { timeout: 300000 });
      }
    },
    fill: async (page) => {
      const content = fs.readFileSync(path.join(CONTENT_DIR, 'v2ex.md'), 'utf-8');
      const title = 'bl — 终端词典客户端，支持离线词典、LLM 翻译、语法解析';
      let inBody = false, body = [];
      for (const line of content.split('\n')) {
        if (!inBody && (line.startsWith('### 正文') || line.startsWith('## 正文'))) { inBody = true; continue; }
        if (inBody) { if (line.startsWith('---') || line.startsWith('## 掘金')) break; body.push(line); }
      }
      await page.fill('input[name="title"]', title);
      await page.fill('textarea[name="content"]', body.join('\n').trim());
      console.log('✅ V2EX 已填好');
    }
  },
  'twitter': {
    name: 'Twitter/X',
    check: async (page) => {
      if (page.url().includes('login')) {
        console.log('\n⚠️  需要登录 Twitter');
        await page.waitForURL('**/compose/**', { timeout: 300000 });
      }
    },
    fill: async (page) => {
      console.log('\n📋 Twitter 线程内容已打印在终端, 逐条粘贴到浏览器');
    }
  },
};

async function main() {
  const platform = process.argv[2];
  if (!platform || !PLATFORMS[platform]) {
    console.log('用法: node publish-helper.js <platform>');
    console.log('平台:', Object.keys(PLATFORMS).join(', '));
    process.exit(1);
  }

  const info = PLATFORMS[platform];
  console.log('🚀 ' + info.name);
  console.log('  浏览器窗口打开后, 可能需要先登录, 脚本自动等待\n');

  const context = await chromium.launchPersistentContext(
    path.join(DATA_DIR, platform),
    { headless: false }
  );
  const page = context.pages()[0] || await context.newPage();

  try {
    await page.goto(URLS[platform], { waitUntil: 'networkidle' });
    await info.check(page);
    await info.fill(page);
    console.log('\n⏸️  浏览器保持打开, 审核后手动操作, Ctrl+C 退出');
    await new Promise(resolve => process.on('SIGINT', resolve));
  } catch (err) {
    console.error('❌:', err.message);
  }
}

main();
