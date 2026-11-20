// 화면 전체를 그리는 코드. 프로그램(서버)이 계산한 단계 상태와 판정을 받아 와서 보여 주고,
// 사용자가 고친 내용은 잠깐 뒤 자동으로 저장한다.

const TOKEN = document.getElementById('app').dataset.token;
let S = null;          // 프로그램이 보내 준 전체 상태
let P = null;          // 지금 화면에서 고치고 있는 프로젝트 기록
let cur = null;        // 보고 있는 단계 번호
let evFilter = 'todo'; // 모은 글 목록의 걸러 보기
let live = [];         // 저장 뒤 다시 그릴 작은 조각들
let saveTimer = null;
let saveChain = Promise.resolve();

// ---------- 기본 도구 ----------

async function api(method, path, body) {
  const res = await fetch(path, {
    method,
    headers: { 'Content-Type': 'application/json', 'X-App-Token': TOKEN },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  let data = {};
  try { data = await res.json(); } catch (e) { /* 내용 없는 응답 */ }
  if (!res.ok) throw new Error(data.error || '프로그램이 응답하지 않았습니다.');
  return data;
}

// h('div', {class:'x', onclick: fn}, '글자', 다른요소) 처럼 화면 요소를 만든다.
function h(tag, attrs, ...kids) {
  const el = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs || {})) {
    if (v === null || v === undefined || v === false) continue;
    if (k.startsWith('on')) el.addEventListener(k.slice(2), v);
    else if (k === 'value') el.value = v;
    else if (k === 'checked') el.checked = v;
    else el.setAttribute(k, v === true ? '' : v);
  }
  for (const kid of kids.flat(Infinity)) {
    if (kid === null || kid === undefined || kid === false) continue;
    el.append(kid.nodeType ? kid : document.createTextNode(String(kid)));
  }
  return el;
}

// el 안의 내용을 통째로 바꾼다. 목록(배열)은 펼치고, 빈 값은 건너뛴다.
function put(el, ...kids) {
  const nodes = kids.flat(Infinity).filter(k => k !== null && k !== undefined && k !== false)
    .map(k => k.nodeType ? k : document.createTextNode(String(k)));
  el.replaceChildren(...nodes);
}

let toastTimer = null;
function toast(msg, isError) {
  const t = document.getElementById('toast');
  t.textContent = msg;
  t.className = 'show' + (isError ? ' err' : '');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => { t.className = ''; }, isError ? 6000 : 2600);
}

function newId() { return Math.random().toString(36).slice(2, 10); }
function today() { const d = new Date(); return d.getFullYear() + '-' + String(d.getMonth() + 1).padStart(2, '0') + '-' + String(d.getDate()).padStart(2, '0'); }
function won(n) { return Math.round(n).toLocaleString('ko-KR'); }

function applyState(st) {
  S = st;
  P = st.project;
}

// 고친 내용을 0.5초 뒤에 저장한다. 계속 입력하는 동안에는 미룬다.
function touch() {
  clearTimeout(saveTimer);
  saveTimer = setTimeout(() => { flush(); }, 500);
}

// 지금 바로 저장한다. 저장이 끝나면 단계 상태와 판정만 새로 받아 조용히 고쳐 그린다.
function flush() {
  clearTimeout(saveTimer);
  saveTimer = null;
  const snapshot = JSON.stringify(P);
  saveChain = saveChain.then(async () => {
    try {
      const data = await api('PUT', '/api/project', JSON.parse(snapshot));
      S.stages = data.state.stages;
      S.judge = data.state.judge;
      S.customerProblems = data.state.customerProblems;
      renderRail();
      live.forEach(fn => fn());
    } catch (e) {
      toast('저장하지 못했습니다. ' + e.message, true);
    }
  });
  return saveChain;
}

// 프로그램에 일을 시키는 버튼용. 누르는 동안 버튼을 잠그고, 끝나면 화면을 통째로 다시 그린다.
async function run(btn, busyText, work) {
  const old = btn.textContent;
  btn.disabled = true;
  btn.textContent = busyText;
  try {
    await flush();
    await work();
  } catch (e) {
    toast(e.message, true);
  } finally {
    btn.disabled = false;
    btn.textContent = old;
  }
}

async function copyText(text) {
  try {
    await navigator.clipboard.writeText(text);
  } catch (e) {
    const ta = h('textarea', { value: text });
    document.body.append(ta);
    ta.select();
    document.execCommand('copy');
    ta.remove();
  }
}

function openModal(...kids) {
  const m = document.getElementById('modal');
  put(m, h('div', { class: 'dialog', role: 'dialog', 'aria-modal': 'true' }, kids));
  m.hidden = false;
  m.onclick = (e) => { if (e.target === m) closeModal(); };
}
function closeModal() { document.getElementById('modal').hidden = true; }
document.addEventListener('keydown', (e) => { if (e.key === 'Escape') closeModal(); });

// 지시문을 보여 주고 복사해 주는 창.
async function showPrompt(kind, task, params, where) {
  const data = await api('POST', '/api/prompt', { kind, task, params: params || {} });
  await copyText(data.text);
  openModal(
    h('h3', {}, '지시문을 복사했습니다'),
    h('p', { class: 'hint' }, where),
    h('textarea', { class: 'tall', readonly: true, value: data.text }),
    h('div', { class: 'actions' },
      h('button', { class: 'btn', onclick: async () => { await copyText(data.text); toast('다시 복사했습니다.'); } }, '다시 복사'),
      h('button', { class: 'btn quiet', onclick: closeModal }, '닫기')),
  );
}

// 입력칸 하나. obj[key]와 묶여서, 고치면 자동 저장된다.
function field(label, hint, obj, key, opts = {}) {
  const tag = opts.area ? 'textarea' : 'input';
  const input = h(tag, {
    type: opts.area ? null : (opts.type || 'text'),
    class: opts.tall ? 'tall' : null,
    placeholder: opts.placeholder || null,
    value: obj[key] === 0 && opts.type === 'number' ? '' : (obj[key] ?? ''),
    oninput: () => {
      obj[key] = opts.type === 'number' ? (parseFloat(input.value) || 0) : input.value;
      touch();
      if (opts.oninput) opts.oninput();
    },
  });
  return h('label', { class: 'field' }, h('span', {}, label), hint ? h('small', {}, hint) : null, input);
}

function checklist(items) {
  return h('ul', { class: 'checks' }, items.map(it => h('li', {},
    h('input', { type: 'checkbox', id: 'c-' + it.id, checked: !!P.checks[it.id], onchange: (e) => { P.checks[it.id] = e.target.checked; flush(); } }),
    h('label', { for: 'c-' + it.id }, h('b', {}, it.title), h('span', { class: 'd' }, it.desc), it.extra || null),
  )));
}

// AI에게 맡기는 작업 상자. 키가 있으면 프로그램 안에서 답을 받고, 없으면 지시문을 복사해 준다.
function aiBox({ button, task, params, get, set, placeholder, where }) {
  const out = h('textarea', { class: 'tall', placeholder, value: get() || '', oninput: () => { set(out.value); touch(); } });
  const go = h('button', { class: 'btn' }, S.settings.hasAiKey ? button : button + ' (지시문 복사)');
  go.onclick = () => run(go, S.settings.hasAiKey ? 'AI가 쓰는 중…' : '만드는 중…', async () => {
    const p = params ? params() : {};
    if (S.settings.hasAiKey) {
      const data = await api('POST', '/api/ai/run', { task, params: p });
      out.value = data.text;
      set(data.text);
      await flush();
      toast('받아 왔습니다. 읽어 보고 고쳐 쓰세요.');
    } else {
      await showPrompt('ai', task, p, where || '쓰던 AI(ChatGPT, Claude 등)에 붙여넣고, 받은 답을 아래 칸에 붙여넣으세요.');
    }
  });
  const copyOnly = S.settings.hasAiKey
    ? h('button', { class: 'link', onclick: () => showPrompt('ai', task, params ? params() : {}, '이 지시문을 다른 AI에 직접 붙여넣어 쓸 수도 있습니다.').catch(e => toast(e.message, true)) }, '지시문만 보기')
    : h('button', { class: 'link', onclick: openSettings }, 'AI 키를 넣으면 프로그램 안에서 바로 받습니다');
  return h('div', {}, h('div', { class: 'actions' }, go, copyOnly), h('div', { class: 'out' }, out));
}

function toolPromptButton(label, task, where) {
  const b = h('button', { class: 'btn quiet small' }, label);
  b.onclick = () => run(b, '만드는 중…', () => showPrompt('tool', task, {}, where || '서비스를 만들 때 쓰는 AI 코딩 도구에 붙여넣으세요.'));
  return b;
}

// ---------- 왼쪽 단계 목록 ----------

function renderRail() {
  const rail = document.getElementById('rail');
  put(rail, 
    h('div', { class: 'rail-head' },
      h('h1', {}, '바이브 창업 코치'),
      h('p', {}, P.name || '프로젝트 이름을 정해 주세요')),
    h('ol', { class: 'stages' }, S.stages.map(st => h('li', {},
      h('button', { class: 'stage-btn', 'data-status': st.status, 'aria-current': st.id === cur ? 'true' : null, onclick: () => { cur = st.id; render(); } },
        h('span', { class: 'num' }, st.id + 1),
        h('span', { class: 'name' }, st.title),
        st.status === 'done' ? h('span', { class: 'mini-seal', title: '통과' }, '통과')
          : st.status === 'now' ? h('span', { class: 'now-tag' }, '지금') : h('span', {}))))),
    h('div', { class: 'rail-foot' },
      h('button', { onclick: openSettings }, '설정'),
      h('p', { class: 'keys' }, 'AI 키 ' + (S.settings.hasAiKey ? '있음' : '없음') + ', 네이버 검색 키 ' + (S.settings.hasNaver ? '있음' : '없음'))),
  );
}

// ---------- 오른쪽 작업 면 ----------

function render() {
  live = [];
  if (cur === null) cur = (S.stages.find(s => s.status === 'now') || S.stages[0]).id;
  renderRail();
  const st = () => S.stages[cur];
  const state = h('p', { class: 'state' });
  const gate = h('p', { class: 'gate' });
  const lock = h('p', { class: 'lockline' });
  const paint = () => {
    state.textContent = st().note;
    gate.className = 'gate' + (st().status === 'done' ? ' done' : '');
    gate.textContent = st().status === 'done' ? '이 단계는 통과했습니다.' : '넘어가려면: ' + st().next;
    lock.hidden = st().status !== 'locked';
    lock.textContent = '앞 단계가 끝나면 이 단계가 "지금 할 일"이 됩니다. 미리 둘러보고 채워 둘 수는 있습니다.';
  };
  live.push(paint);
  paint();
  const body = VIEWS[cur]();
  const work = document.getElementById('work');
  put(work, h('div', { class: 'sheet' },
    h('header', { class: 'stage-head' }, h('h2', {}, (cur + 1) + '. ' + st().title), state, gate),
    lock, body));
  work.scrollTop = 0;
}

function block(title, why, ...kids) {
  return h('section', { class: 'block' }, h('h3', {}, title), why ? h('p', { class: 'why' }, why) : null, kids);
}

const VIEWS = [];

// 1. 고객·문제 정하기
VIEWS[0] = () => {
  const c = P.customer;
  const sentence = h('p', { class: 'sentence' });
  const warn = h('div', { class: 'warn' });
  const paintSentence = () => {
    const g = (v, ph) => v && v.trim() ? v.trim() : '[' + ph + ']';
    sentence.textContent = `${g(c.who, '누구')}은(는) ${g(c.situation, '어떤 상황')}에서 ${g(c.problem, '어떤 문제')}을(를) 겪는다. 그래서 ${g(c.solution, '무엇')}을(를) 만든다.`;
  };
  const paintWarn = () => {
    const probs = S.customerProblems || [];
    warn.hidden = probs.length === 0;
    put(warn, h('ul', {}, probs.map(t => h('li', {}, t))));
  };
  live.push(paintWarn);
  paintSentence(); paintWarn();
  const o = { oninput: paintSentence };
  return h('div', {},
    block('프로젝트', null,
      field('프로젝트 이름', null, P, 'name', { placeholder: '예: 육아용품 최저가 알림', oninput: renderRailName })),
    block('고객 한 문장', '네 칸을 채우면 한 문장이 됩니다. "누구"가 좁을수록 그 사람들이 쓴 글을 찾기 쉽고, 나중에 글을 올릴 곳도 정해집니다.',
      field('누구', '나이대가 아니라 처한 상황으로 적습니다.', c, 'who', { placeholder: '예: 돌 전 아기를 키우는 부모', ...o }),
      field('어떤 상황에서', null, c, 'situation', { placeholder: '예: 기저귀와 분유를 살 때마다', ...o }),
      field('어떤 문제를 겪나', null, c, 'problem', { placeholder: '예: 어느 쇼핑몰이 지금 가장 싼지 서너 곳을 직접 비교해야 하는', ...o }),
      field('그래서 무엇을 만드나', null, c, 'solution', { placeholder: '예: 그 두 품목의 오늘 최저가만 보여 주는 페이지', ...o }),
      sentence, warn),
    block('더 좁히기', '문장이 넓다는 경고가 뜨거나 어떻게 좁힐지 모르겠으면 제안을 받아 보세요. 다음 단계에서 쓸 검색어도 함께 나옵니다.',
      aiBox({ button: '좁히는 방법 제안받기', task: 'narrow', get: () => P.drafts.narrow, set: v => { P.drafts.narrow = v; }, placeholder: '제안이 여기에 들어옵니다.' })),
  );
};
function renderRailName() { const p = document.querySelector('.rail-head p'); if (p) p.textContent = P.name || '프로젝트 이름을 정해 주세요'; }

// 2. 만들기 전 검증: 사람에게 묻지 않고, 사람들이 이미 쓴 글로 확인한다.
VIEWS[1] = () => {
  const need = S.settings.thresholds.evidence;
  const count = () => {
    let total = 0, classified = 0, strong = 0;
    for (const e of P.evidence) { total++; if (e.experienced !== null && e.experienced !== undefined) { classified++; if (e.experienced && e.tried) strong++; } }
    return { total, classified, strong, todo: total - classified };
  };
  const tally = h('div', {});
  const list = h('div', {});
  const paintTally = () => {
    const n = count();
    put(tally, 
      h('div', { class: 'tally' },
        h('div', { class: 'big' }, n.strong, h('small', {}, ' / ' + need + '건')),
        h('div', { class: 'label' }, '글쓴이가 이 문제를 실제로 겪었고 스스로 해결하려 한 글. 이 숫자가 "문제가 진짜인가"의 근거입니다.')),
      h('div', { class: 'marks' }, Array.from({ length: need }, (_, i) => h('i', { class: i < n.strong ? 'on' : null }))),
      h('p', { class: 'hint' }, `모은 글 ${n.total}건, 분류한 글 ${n.classified}건, 아직 안 본 글 ${n.todo}건`));
  };
  const setClass = (e, exp, tried) => { e.experienced = exp; e.tried = tried; e.by = 'me'; flush(); paintTally(); paintList(); };
  const paintList = () => {
    const shown = P.evidence.filter(e => evFilter === 'all' ? true : evFilter === 'todo' ? (e.experienced === null || e.experienced === undefined) : (e.experienced && e.tried));
    const n = count();
    put(list, 
      h('div', { class: 'filters' },
        [['todo', `아직 안 본 글 ${n.todo}`], ['strong', `겪고 해결하려 한 글 ${n.strong}`], ['all', `전체 ${n.total}`]].map(([k, label]) =>
          h('button', { 'aria-pressed': String(evFilter === k), onclick: () => { evFilter = k; paintList(); } }, label))),
      shown.length === 0 ? h('p', { class: 'hint' }, P.evidence.length === 0 ? '아직 모은 글이 없습니다. 위에서 검색하거나 붙여넣어 주세요.' : '이 조건에 맞는 글이 없습니다.') : null,
      shown.slice(0, 60).map(e => h('div', { class: 'ev' },
        h('h4', {}, e.link ? h('a', { href: e.link, target: '_blank', rel: 'noreferrer' }, e.title) : e.title),
        h('div', { class: 'meta' }, [e.source, e.date, e.keyword ? '검색어: ' + e.keyword : ''].filter(Boolean).join(', ')),
        e.snippet ? h('p', {}, e.snippet) : null,
        e.problem ? h('div', { class: 'ai' }, (e.by === 'ai' ? 'AI가 읽은 문제: ' : '문제: ') + e.problem + (e.quote ? ` / 표현: "${e.quote}"` : '')) : null,
        h('div', { class: 'choice' },
          h('button', { 'aria-pressed': String(e.experienced === true && e.tried === true), onclick: () => setClass(e, true, true) }, '겪었고 해결하려 함'),
          h('button', { 'aria-pressed': String(e.experienced === true && e.tried === false), onclick: () => setClass(e, true, false) }, '겪기만 함'),
          h('button', { 'aria-pressed': String(e.experienced === false), onclick: () => setClass(e, false, false) }, '해당 없음(광고·정보 글)'),
          h('button', { class: 'del', onclick: () => { P.evidence = P.evidence.filter(x => x.id !== e.id); flush(); paintTally(); paintList(); } }, '지우기')))),
      shown.length > 60 ? h('p', { class: 'hint' }, `나머지 ${shown.length - 60}건은 위 글을 분류하면 이어서 보입니다.`) : null);
  };
  paintTally(); paintList();

  const kw = h('input', { type: 'text', placeholder: '예: 기저귀 최저가 비교' });
  const searchBtn = h('button', { class: 'btn' }, '네이버에서 모으기');
  searchBtn.onclick = () => run(searchBtn, '모으는 중…', async () => {
    const data = await api('POST', '/api/collect/naver', { keyword: kw.value });
    applyState(data.state); kw.value = '';
    toast(`${data.found}건을 찾아 새 글 ${data.added}건을 더했습니다.`);
    render();
  });
  const paste = h('textarea', { placeholder: '직접 찾은 글의 제목과 내용을 붙여넣으세요. 글과 글 사이는 빈 줄로 띄웁니다. 주소를 같이 붙여넣으면 링크로 저장됩니다.' });
  const pasteBtn = h('button', { class: 'btn quiet' }, '붙여넣은 글 더하기');
  pasteBtn.onclick = () => run(pasteBtn, '더하는 중…', async () => {
    const data = await api('POST', '/api/collect/paste', { text: paste.value });
    applyState(data.state);
    toast(`새 글 ${data.added}건을 더했습니다.`);
    render();
  });
  const classifyBtn = h('button', { class: 'btn' }, 'AI로 한꺼번에 분류하기');
  classifyBtn.onclick = () => run(classifyBtn, 'AI가 읽는 중…', async () => {
    const data = await api('POST', '/api/ai/classify');
    applyState(data.state);
    toast(data.done === 0 ? '분류할 글이 없습니다.' : `${data.done}건을 분류했습니다.` + (data.remaining > 0 ? ` 남은 ${data.remaining}건은 한 번 더 누르세요.` : ''));
    render();
  });

  return h('div', {},
    block('지금까지의 근거', null, tally),
    block('고객이 쓴 글 모으기', '사람에게 직접 묻지 않습니다. 같은 문제를 겪는 사람들이 지식iN, 카페, 블로그에 이미 써 둔 글을 모아 읽습니다. 제목과 요약만 가져옵니다.',
      S.settings.hasNaver
        ? h('div', { class: 'row' }, h('label', { class: 'field grow' }, h('span', {}, '검색어'), h('small', {}, '고객이 이 문제를 검색할 때 쓸 말로 넣습니다. 여러 번 바꿔 가며 모으세요.'), kw), h('div', { class: 'field' }, searchBtn))
        : h('p', { class: 'hint' }, '네이버 검색 키를 넣으면 검색어만으로 글을 모아 옵니다. ', h('button', { class: 'link', onclick: openSettings }, '설정에서 키 넣기'), ' 키가 없어도 아래에 직접 붙여넣어 쓸 수 있습니다.'),
      P.keywords.length ? h('p', { class: 'hint' }, '지금까지 쓴 검색어: ' + P.keywords.join(', ')) : null,
      h('label', { class: 'field' }, h('span', {}, '직접 붙여넣기'), paste),
      h('div', { class: 'actions' }, pasteBtn)),
    block('글 읽고 분류하기', '글마다 글쓴이가 이 문제를 실제로 겪는지, 스스로 해결하려 했는지를 표시합니다. 광고와 업체 글은 "해당 없음"입니다.',
      S.settings.hasAiKey ? h('div', { class: 'actions' }, classifyBtn, h('span', { class: 'hint' }, 'AI의 분류가 틀린 글은 아래에서 직접 고치세요.'))
        : h('p', { class: 'hint' }, 'AI 키가 있으면 한꺼번에 분류해 줍니다. ', h('button', { class: 'link', onclick: openSettings }, '설정에서 키 넣기'), ' 키가 없으면 아래에서 하나씩 눌러 분류합니다.'),
      list),
    block('문제와 표현 정리', '분류한 글에서 자주 나온 문제, 사람들이 지금 쓰는 해결 방법, 첫 화면 문구로 쓸 표현을 뽑습니다.',
      aiBox({ button: '정리하기', task: 'summary', get: () => P.drafts.summary, set: v => { P.drafts.summary = v; }, placeholder: '정리한 내용이 여기에 들어옵니다.' })),
  );
};

// 3. 기획·AI 지시
VIEWS[2] = () => h('div', {},
  block('한 장 기획서', '고객 한 문장과 모은 글을 바탕으로 첫 버전의 범위를 정합니다. 받은 뒤 직접 읽고 줄이세요. 첫 버전에 넣는 것은 3개 이하가 좋습니다.',
    aiBox({ button: '한 장 기획서 만들기', task: 'plan', get: () => P.plan, set: v => { P.plan = v; }, placeholder: '기획서가 여기에 들어옵니다. 직접 써도 됩니다.' })),
  block('코딩 도구에 넘기기', '기획서를 AI 코딩 도구에 붙여넣을 지시문으로 바꿉니다. 한 번에 다 만들지 않고 순서를 나눠 진행하게 시킵니다.',
    h('div', { class: 'actions' }, toolPromptButton('만들기 지시문 복사', 'plan'))),
);

// 4. 만들기
VIEWS[3] = () => h('div', {},
  block('만드는 동안 확인할 것', '만드는 방법은 이미 좋은 자료가 많습니다. 여기서는 나중에 되돌리기 어려운 것만 확인합니다.',
    checklist([
      { id: 'build.small', title: '기획서의 "첫 버전에 넣는 것"만 만들었다', desc: '만들다 떠오른 기능은 기획서의 "넣지 않는 것"에 적어 두고 넘어갑니다.' },
      { id: 'build.save', title: '잘 되는 상태를 저장해 두었다', desc: '코딩 도구에 "지금 상태를 저장(커밋)해 줘"라고 시킵니다. 다음 수정이 망가뜨려도 돌아올 수 있습니다.' },
      { id: 'build.secret', title: '비밀 키가 코드나 화면에 드러나 있지 않다', desc: '코딩 도구에 "비밀 키나 비밀번호가 코드나 브라우저에 노출된 곳이 있는지 찾아서 고쳐 줘"라고 시킵니다.' },
      { id: 'build.mobile', title: '휴대폰에서 직접 열어 끝까지 써 봤다', desc: '커뮤니티에서 들어오는 사람은 대부분 휴대폰으로 봅니다.' },
    ])),
  block('서비스 주소', '다른 사람이 열 수 있는 주소가 생기면 넣으세요. 다음 단계의 점검에 씁니다.',
    field('주소', null, P, 'serviceUrl', { type: 'url', placeholder: 'https://' })),
);

// 5. 출시 준비
VIEWS[4] = () => {
  const btn = h('button', { class: 'btn' }, P.siteCheck ? '다시 점검하기' : '첫 화면 점검하기');
  btn.onclick = () => run(btn, '열어 보는 중…', async () => {
    const data = await api('POST', '/api/sitecheck');
    applyState(data.state);
    render();
  });
  const sc = P.siteCheck;
  const failed = sc ? sc.results.filter(r => !r.pass).length : 0;
  return h('div', {},
    block('첫 화면 점검', '프로그램이 서비스 주소를 직접 열어, 처음 온 사람과 검색·공유에 필요한 기본을 갖췄는지 봅니다.',
      field('서비스 주소', null, P, 'serviceUrl', { type: 'url', placeholder: 'https://' }),
      h('div', { class: 'actions' }, btn, sc && failed > 0 ? toolPromptButton(`고칠 점 ${failed}개 지시문 복사`, 'sitefix') : null),
      sc ? h('div', { style: 'margin-top:14px' },
        sc.error ? h('div', { class: 'warn' }, sc.error) : h('p', { class: 'hint' }, `${sc.at}에 점검함. ${sc.results.length}개 중 ${sc.results.length - failed}개 통과.`),
        sc.results.map(r => h('div', { class: 'result ' + (r.pass ? 'pass' : 'fail') },
          h('span', { class: 'mark' }, r.pass ? '통과' : '고치기'),
          h('span', {}, r.label, h('span', { class: 'd' }, r.detail))))) : null),
    block('사람에게 묻지 않고 반응을 보는 준비', '방문 기록과 익명 한 줄 설문이 있으면 누구에게도 직접 묻지 않고 판정할 수 있습니다.',
      checklist([
        { id: 'launch.analytics', title: '방문 기록 도구를 붙였다', desc: '하루 방문자, 다시 온 방문자, 어디서 왔는지를 볼 수 있어야 합니다.', extra: h('div', { class: 'actions' }, toolPromptButton('붙이는 지시문 복사', 'analytics')) },
        { id: 'launch.survey', title: '익명 한 줄 설문을 붙였다', desc: '"이게 도움이 됐나요?" 버튼 두 개와 선택 입력칸 하나. 이름과 연락처는 받지 않습니다.', extra: h('div', { class: 'actions' }, toolPromptButton('붙이는 지시문 복사', 'survey')) },
        { id: 'launch.policy', title: '공개 전에 확인할 것을 확인했다', desc: '개인정보를 받는다면 처리방침(개인정보보호위원회 안내), 수수료를 받는 링크가 있다면 고지 문구(공정거래위원회 지침과 제휴 프로그램 약관), 수익이 생기면 사업자등록(국세청 상담). 이 프로그램은 법률 판단을 하지 않습니다. 각 기관의 안내를 직접 확인하세요.' },
      ])),
  );
};

// 6. 알리기·첫 사용자: 글 올리기와 댓글 답하기만 한다. 누구에게도 따로 연락하지 않는다.
VIEWS[5] = () => {
  const places = h('div', {});
  const posts = h('div', {});
  const paintPlaces = () => put(places, 
    P.places.length === 0 ? h('p', { class: 'hint' }, '아직 정한 곳이 없습니다. 2단계에서 모은 글이 많이 올라온 카페나 게시판부터 넣어 보세요.') : null,
    P.places.map(pl => {
      const base = (P.serviceUrl || 'https://내서비스주소').replace(/\/$/, '');
      const tag = 'c-' + pl.id.slice(0, 4); // 방문 기록 도구에 이 이름으로 찍힌다
      const link = base + '/?utm_source=' + tag;
      const draftBtn = h('button', { class: 'btn small' }, S.settings.hasAiKey ? '이곳에 올릴 글 초안 만들기' : '글 초안 지시문 복사');
      draftBtn.onclick = () => run(draftBtn, '쓰는 중…', async () => {
        if (!pl.rulesRead) throw new Error('먼저 그곳의 공지와 규칙을 읽고 "규칙을 읽었다"에 표시해 주세요.');
        if (S.settings.hasAiKey) {
          const data = await api('POST', '/api/ai/run', { task: 'post', params: { placeId: pl.id } });
          P.posts.push({ id: newId(), placeId: pl.id, date: today(), title: '', body: data.text, posted: false, views: 0, comments: 0, visits: 0, deleted: false });
          await flush(); paintPosts(); toast('아래 "올린 글과 결과"에 초안을 넣었습니다. 내 말투로 고쳐서 올리세요.');
        } else {
          P.posts.push({ id: newId(), placeId: pl.id, date: today(), title: '', body: '', posted: false, views: 0, comments: 0, visits: 0, deleted: false });
          await flush(); paintPosts();
          await showPrompt('ai', 'post', { placeId: pl.id }, '쓰던 AI에 붙여넣고, 받은 초안을 아래 "올린 글과 결과"의 빈 칸에 붙여넣으세요.');
        }
      });
      return h('div', { class: 'place' },
        field('곳 이름', null, pl, 'name', { placeholder: '예: ○○ 육아 카페 자유게시판' }),
        field('그곳의 규칙', '공지를 직접 읽고, 홍보·링크에 대한 내용을 옮겨 적습니다.', pl, 'rules', { area: true, placeholder: '예: 홍보 글은 [홍보] 게시판에만. 외부 링크는 운영진 승인 필요.' }),
        h('label', { class: 'inline' }, h('input', { type: 'checkbox', checked: pl.rulesRead, onchange: e => { pl.rulesRead = e.target.checked; flush(); } }), '규칙을 읽었다'),
        h('label', { class: 'inline' }, h('input', { type: 'checkbox', checked: pl.linkAllowed, onchange: e => { pl.linkAllowed = e.target.checked; flush(); paintPlaces(); } }), '내 서비스 링크를 올려도 되는 곳이다'),
        pl.linkAllowed ? h('p', { class: 'hint' }, '이곳에 올릴 때 쓸 링크: ', h('b', {}, link), ' ', h('button', { class: 'link', onclick: async () => { await copyText(link); toast('링크를 복사했습니다.'); } }, '복사'), ` (방문 기록 도구에서 "${tag}"로 들어온 방문이 이 글의 방문 수입니다.)`)
          : h('p', { class: 'hint' }, '링크를 올릴 수 없는 곳에서는 서비스 이야기 없이 질문 글만 올립니다. 사람들의 답이 근거가 됩니다.'),
        h('div', { class: 'actions' }, draftBtn, h('button', { class: 'link', onclick: () => { P.places = P.places.filter(x => x.id !== pl.id); flush(); paintPlaces(); } }, '이곳 지우기')));
    }));
  const paintPosts = () => put(posts, 
    P.posts.length === 0 ? h('p', { class: 'hint' }, '아직 글이 없습니다. 위에서 올릴 곳을 정하고 초안을 만드세요.') : null,
    P.posts.map(po => {
      const pl = P.places.find(x => x.id === po.placeId);
      const num = (label, key) => h('label', {}, label, h('input', { type: 'number', min: '0', value: po[key] || '', oninput: e => { po[key] = parseInt(e.target.value, 10) || 0; touch(); } }));
      return h('div', { class: 'post' },
        h('p', { class: 'hint', style: 'margin:0 0 6px' }, (pl ? pl.name : '(지운 곳)') + ', ' + po.date),
        field('글', '초안을 그대로 올리지 말고 내 말투로 고칩니다. 만든 사람이 나라는 것을 밝힙니다.', po, 'body', { area: true }),
        h('div', { class: 'nums' },
          h('label', { class: 'inline' }, h('input', { type: 'checkbox', checked: po.posted, onchange: e => { po.posted = e.target.checked; flush(); } }), '올렸다'),
          num('조회 수', 'views'), num('댓글 수', 'comments'), num('이 글로 들어온 방문', 'visits'),
          h('label', { class: 'inline' }, h('input', { type: 'checkbox', checked: po.deleted, onchange: e => { po.deleted = e.target.checked; flush(); } }), '삭제당했다'),
          h('button', { class: 'link', onclick: () => { P.posts = P.posts.filter(x => x.id !== po.id); flush(); paintPosts(); } }, '지우기')));
    }));
  paintPlaces(); paintPosts();
  const comment = { text: '' };
  return h('div', {},
    block('올릴 곳', '고객이 모여 있는 곳을 정하고, 그곳의 규칙을 먼저 읽습니다. 같은 글을 여러 곳에 그대로 올리지 않습니다. 규칙이 금지한 곳에는 올리지 않습니다.',
      places,
      h('div', { class: 'actions' }, h('button', { class: 'btn quiet', onclick: () => { P.places.push({ id: newId(), name: '', rules: '', linkAllowed: false, rulesRead: false }); flush(); paintPlaces(); } }, '곳 더하기'))),
    block('올린 글과 결과', '올린 다음 날 조회 수, 댓글 수, 방문 수를 적습니다. 반응이 없는 글이 더 흔합니다. 숫자가 작아도 그대로 적으세요.',
      posts),
    block('댓글에 답하기', '댓글이 달리면 그날 안에 답합니다. 댓글을 붙여넣으면 답글 초안을 만들어 줍니다.',
      field('달린 댓글', null, comment, 'text', { area: true, placeholder: '댓글 내용을 붙여넣으세요.' }),
      aiBox({ button: '답글 초안 만들기', task: 'reply', params: () => ({ comment: comment.text }), get: () => P.drafts.reply, set: v => { P.drafts.reply = v; }, placeholder: '답글 초안이 여기에 들어옵니다.' })),
  );
};

// 7. 검증·판정
VIEWS[6] = () => {
  const box = h('div', {});
  const paint = () => {
    const j = S.judge;
    const empty = j.verdict === '근거 부족';
    put(box, 
      h('div', { class: 'verdict' },
        h('div', { class: 'seal' + (empty ? ' empty' : '') }, j.verdict.replace(': ', ':\n')),
        h('p', {}, j.nextStep)),
      h('table', { class: 'judge' },
        h('thead', {}, h('tr', {}, h('th', {}, '질문'), h('th', {}, '세는 것'), h('th', {}, '지금 / 기준'), h('th', {}, '결과'))),
        h('tbody', {}, j.questions.map(q => h('tr', {},
          h('th', {}, q.key + '. ' + q.label),
          h('td', {}, h('small', {}, q.detail)),
          h('td', { class: 'n' }, q.have + ' / ' + q.need),
          h('td', { class: 'p' + (q.pass ? ' yes' : '') }, q.pass ? '통과' : '아직'))))),
      h('p', { class: 'hint' }, '기준 숫자는 출발점으로 잡은 값입니다. 설정에서 바꿀 수 있고, 바꾼다면 왜 바꾸는지 10단계 기록에 적어 두세요.'));
  };
  live.push(paint); paint();
  const sv = P.survey;
  const comments = { text: sv.comments.join('\n') };
  return h('div', {},
    block('판정', '사람에게 직접 묻지 않고 얻은 숫자 세 가지로 정합니다. 숫자가 기준에 못 미치는데 느낌만으로 "계속"을 고르지 않게 하려는 장치입니다.', box),
    block('익명 설문과 재방문 숫자', '5단계에서 붙인 익명 설문의 결과와 방문 기록 도구의 재방문자 수를 옮겨 적습니다.',
      h('div', { class: 'nums' },
        h('label', {}, '"도움 됐어요" 수', h('input', { type: 'number', min: '0', value: sv.helpful || '', oninput: e => { sv.helpful = parseInt(e.target.value, 10) || 0; touch(); } })),
        h('label', {}, '"아니요" 수', h('input', { type: 'number', min: '0', value: sv.notHelpful || '', oninput: e => { sv.notHelpful = parseInt(e.target.value, 10) || 0; touch(); } })),
        h('label', {}, '다시 온 사람 수', h('input', { type: 'number', min: '0', value: P.returning || '', oninput: e => { P.returning = parseInt(e.target.value, 10) || 0; touch(); } }))),
      h('div', { style: 'margin-top:12px' },
        field('설문에 적힌 한 줄들', '한 줄에 하나씩 옮겨 적습니다. 8단계에서 고칠 점을 고를 때 씁니다.', comments, 'text', { area: true, oninput: () => { sv.comments = comments.text.split('\n').map(s => s.trim()).filter(Boolean); } }))),
  );
};

// 8. 고치기·다시 오게 하기
VIEWS[7] = () => h('div', {},
  block('설문에 적힌 불편', null,
    P.survey.comments.length ? h('ul', {}, P.survey.comments.map(c => h('li', {}, c))) : h('p', { class: 'hint' }, '아직 적힌 한 줄이 없습니다. 7단계에서 설문 결과를 옮겨 적으면 여기에 보입니다.')),
  block('한 번에 하나만', '기능을 더 만들고 싶어지는 단계입니다. 판정이 "수정: 서비스"일 때만 서비스를 고치고, 그때도 하나만 고칩니다.',
    checklist([
      { id: 'improve.fixone', title: '가장 많이 나온 불편 하나를 골라 고쳤다', desc: '고친 뒤 같은 곳에 "의견 주신 덕분에 이렇게 고쳤습니다"라고 알립니다. 새 글이 아니라 원래 글의 댓글로 답니다.' },
      { id: 'improve.return', title: '다시 오게 할 장치 하나를 정했다', desc: '예: 값이 바뀌면 알려 주는 알림, 매주 같은 요일에 새로 올라오는 정리, 즐겨찾기 해 둘 이유. 광고 수익은 다시 오는 사람 수에 달려 있습니다.' },
    ])),
);

// 9. 수익 계산
VIEWS[8] = () => {
  const r = P.revenue;
  const out = h('div', { class: 'calc' });
  const paint = () => {
    const views = r.monthlyVisitors * (r.pagesPerVisit || 1);
    const income = views / 1000 * r.rpm;
    const perVisit = (r.pagesPerVisit || 1) * r.rpm / 1000;
    const needed = perVisit > 0 ? r.target / perVisit : 0;
    put(out, 
      r.monthlyVisitors > 0 && r.rpm > 0
        ? h('p', {}, '지금 방문자 수로 예상되는 월 광고 수익은 ', h('b', {}, won(income) + '원'), '입니다.')
        : h('p', { class: 'hint' }, '방문자 수와 1,000번 노출당 수익을 넣으면 계산됩니다.'),
      r.target > 0 && perVisit > 0
        ? h('p', {}, '목표 ' + won(r.target) + '원을 벌려면 한 달 방문이 ', h('b', {}, won(needed) + '번'), ' 필요합니다.' + (r.monthlyVisitors > 0 ? ` 지금의 ${(needed / r.monthlyVisitors).toFixed(1)}배입니다.` : ''))
        : null);
  };
  paint();
  const o = { type: 'number', oninput: paint };
  return h('div', {},
    block('광고 수익 계산', '"방문자 몇 명이면 월 얼마"를 직접 계산해 봅니다. 목표에 필요한 방문 수가 비현실적이면 수익 모델을 바꿀 때입니다.',
      field('한 달 방문 수', '방문 기록 도구에 찍힌 숫자를 넣습니다.', r, 'monthlyVisitors', o),
      field('한 번 방문에 보는 페이지 수', '모르면 1을 넣습니다.', r, 'pagesPerVisit', o),
      field('1,000번 노출당 수익(원)', '광고 서비스의 내 계정 화면에 찍힌 실제 값을 넣습니다. 아직 광고를 달지 않았다면 낮게 잡은 값으로 계산해 보세요. 이 프로그램은 기본값을 제시하지 않습니다. 믿을 만한 공개 수치를 찾지 못했기 때문입니다.', r, 'rpm', o),
      field('목표 월 수익(원)', null, r, 'target', o),
      out),
    block('광고 말고', null,
      h('p', { class: 'why' }, '광고는 방문이 아주 많아야 의미가 생깁니다. 방문이 적어도 되는 방식(특정 사람에게 값을 받는 유료 기능, 제휴 수수료, 후원)도 같은 표로 따져 보세요. 제휴 링크를 쓴다면 수수료를 받는다는 사실을 글 첫머리에 밝혀야 합니다. 문구와 위치는 공정거래위원회 지침과 해당 제휴 프로그램 약관을 확인하세요.')),
  );
};

// 10. 운영·기록
VIEWS[9] = () => {
  const entry = { text: '' };
  const list = h('ul', { class: 'loglist' });
  const paint = () => put(list, P.log.slice().reverse().map(l => h('li', {}, h('time', {}, l.date), l.text)));
  paint();
  const input = h('textarea', { placeholder: '이번 주에 한 일, 나온 숫자, 다음 주에 할 일 하나.', oninput: e => { entry.text = e.target.value; } });
  return h('div', {},
    block('혼자 오래 하기 위한 두 가지', null,
      checklist([
        { id: 'ops.cost', title: '한 달 비용 상한을 정했다', desc: 'AI 도구 구독료, AI 사용료, 서버 비용을 더해 한 달에 쓸 최대 금액을 정하고, 각 서비스에 사용 한도를 걸어 둡니다.' },
        { id: 'ops.backup', title: '코드와 데이터가 내 컴퓨터 밖에도 있다', desc: '코드는 GitHub 같은 저장소에, 데이터는 내려받은 사본을 주기적으로 보관합니다.' },
      ])),
    block('주간 기록', '반응이 없는 주에도 한 줄을 남깁니다. 석 달 뒤 무엇이 통했는지 알 수 있는 유일한 자료가 됩니다.',
      input,
      h('div', { class: 'actions' }, h('button', { class: 'btn', onclick: () => { if (!entry.text.trim()) return; P.log.push({ date: today(), text: entry.text.trim() }); entry.text = ''; input.value = ''; flush(); paint(); } }, '기록 남기기')),
      list),
  );
};

// ---------- 설정 ----------

function openSettings() {
  const v = { aiKey: '', aiModel: S.settings.aiModel, naverId: '', naverSecret: '' };
  const th = { ...S.settings.thresholds };
  const f = (label, hint, obj, key, type, ph) => {
    const i = h('input', { type: type || 'text', placeholder: ph || null, value: obj[key] ?? '', oninput: () => { obj[key] = type === 'number' ? (parseInt(i.value, 10) || 0) : i.value; } });
    return h('label', { class: 'field' }, h('span', {}, label), hint ? h('small', {}, hint) : null, i);
  };
  const saveBtn = h('button', { class: 'btn' }, '설정 저장');
  saveBtn.onclick = () => run(saveBtn, '저장 중…', async () => {
    const data = await api('POST', '/api/settings', { ...v, thresholds: th });
    applyState(data.state);
    closeModal(); render(); toast('설정을 저장했습니다.');
  });
  openModal(
    h('h3', {}, '설정'),
    h('p', { class: 'hint' }, '키는 이 컴퓨터의 기록 폴더에만 저장되고 다른 곳으로 보내지 않습니다. 이미 넣은 키는 화면에 다시 보여 주지 않습니다. 지우려면 칸에 - 를 넣고 저장하세요.'),
    h('fieldset', {}, h('legend', {}, 'AI'),
      f('Claude API 키', S.settings.hasAiKey ? '저장된 키가 있습니다. 바꿀 때만 새로 넣으세요.' : '넣으면 글 분류와 초안 작성을 프로그램 안에서 바로 합니다. 없으면 지시문을 복사해 쓰던 AI에 붙여넣는 방식으로 동작합니다. 사용료는 키를 발급한 계정에 청구됩니다.', v, 'aiKey', 'password', 'sk-ant-...'),
      f('모델 이름', '모르면 그대로 두세요.', v, 'aiModel')),
    h('fieldset', {}, h('legend', {}, '네이버 검색'),
      h('p', { class: 'hint' }, '네이버 개발자 센터에서 애플리케이션을 등록하고 "검색" 사용을 고르면 Client ID와 Secret이 나옵니다. ' + (S.settings.hasNaver ? '저장된 키가 있습니다.' : '없어도 글을 직접 붙여넣어 쓸 수 있습니다.')),
      f('Client ID', null, v, 'naverId'),
      f('Client Secret', null, v, 'naverSecret', 'password')),
    h('fieldset', {}, h('legend', {}, '판정 기준'),
      h('p', { class: 'hint' }, '출발점으로 잡은 값입니다. 내 서비스에 맞게 바꿀 수 있습니다.'),
      h('div', { class: 'nums' },
        f('겪고 해결하려 한 글', null, th, 'evidence', 'number'),
        f('글로 들어온 방문', null, th, 'visits', 'number'),
        f('"도움 됐어요"', null, th, 'helpful', 'number'),
        f('다시 온 사람', null, th, 'return', 'number'))),
    h('p', { class: 'hint' }, '기록이 저장되는 폴더: ' + S.dataDir),
    h('div', { class: 'actions' }, saveBtn, h('button', { class: 'btn quiet', onclick: closeModal }, '닫기')),
  );
}

// ---------- 시작 ----------

api('GET', '/api/state').then(data => { applyState(data.state); render(); })
  .catch(e => { document.getElementById('work').textContent = '프로그램을 시작하지 못했습니다. ' + e.message; });
