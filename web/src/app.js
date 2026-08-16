const state = {
  difficulty: "simple",
  mode: "chinese",
  prompts: [],
  index: 0,
  startedAt: performance.now(),
};

const byId = (id) => document.getElementById(id);

async function request(path, options = {}) {
  const response = await fetch(path, {
    headers: { "Content-Type": "application/json", ...(options.headers || {}) },
    ...options,
  });
  const body = await response.json();
  if (!response.ok) throw new Error(body.error || "请求失败");
  return body;
}

async function loadPrompts() {
  state.prompts = await request(`/api/challenges?difficulty=${state.difficulty}`);
  state.index = 0;
  renderPrompt();
}

function renderPrompt() {
  const current = state.prompts[state.index];
  byId("round-progress").textContent = current ? `第 ${state.index + 1} / ${state.prompts.length} 题` : "暂无题目";
  byId("prompt").textContent = current ? current[state.mode] : "暂无题目";
  byId("answer-input").value = "";
  byId("answer-input").disabled = !current;
  byId("result-line").textContent = "";
  byId("result-line").className = "result-line";
  byId("next-button").classList.remove("is-visible");
  state.startedAt = performance.now();
  if (current) byId("answer-input").focus();
}

async function submitAnswer(event) {
  event.preventDefault();
  const current = state.prompts[state.index];
  if (!current) return;
  const answer = byId("answer-input").value;
  const elapsedMs = Math.max(0, Math.round(performance.now() - state.startedAt));
  try {
    const result = await request("/api/answers", {
      method: "POST",
      body: JSON.stringify({ challengeId: current.challengeId, answer, elapsedMs }),
    });
    const line = byId("result-line");
    line.textContent = result.correct ? `正确，得分 +${result.score}` : `正确拼写：${result.expected}`;
    line.className = `result-line ${result.correct ? "success" : "error"}`;
    byId("answer-input").disabled = true;
    byId("next-button").classList.add("is-visible");
    await refreshStats();
  } catch (error) {
    byId("result-line").textContent = error.message;
    byId("result-line").className = "result-line error";
  }
}

function nextPrompt() {
  state.index = (state.index + 1) % state.prompts.length;
  renderPrompt();
}

async function refreshWrongWords() {
  const rows = await request("/api/wrong-words");
  byId("wrong-body").replaceChildren(...rows.map((item) => row([
    item.chinese,
    item.answer,
    item.expected,
    `${(item.elapsedMs / 1000).toFixed(1)}s`,
  ])));
  byId("wrong-empty").hidden = rows.length > 0;
}

async function refreshStats() {
  const [stats, history] = await Promise.all([request("/api/stats"), request("/api/history")]);
  const rate = stats.attempts ? Math.round((stats.correct / stats.attempts) * 100) : 0;
  byId("header-score").textContent = stats.totalScore;
  byId("stat-attempts").textContent = stats.attempts;
  byId("stat-rate").textContent = `${rate}%`;
  byId("stat-time").textContent = `${(stats.totalElapsedMs / 1000).toFixed(1)}s`;
  byId("stat-score").textContent = stats.totalScore;
  byId("history-body").replaceChildren(...history.map((item) => {
    const result = document.createElement("tr");
    const values = [
      item.expected,
      item.difficulty === "simple" ? "简单" : "中等",
      item.correct ? "正确" : "错误",
      String(item.score),
      `${(item.elapsedMs / 1000).toFixed(1)}s`,
    ];
    values.forEach((value, index) => {
      const cell = document.createElement("td");
      cell.textContent = value;
      if (index === 2) cell.className = item.correct ? "status-good" : "status-bad";
      result.append(cell);
    });
    return result;
  }));
  byId("history-empty").hidden = history.length > 0;
}

function row(values) {
  const result = document.createElement("tr");
  values.forEach((value) => {
    const cell = document.createElement("td");
    cell.textContent = value;
    result.append(cell);
  });
  return result;
}

function formValue(id) {
  return Object.fromEntries(new FormData(byId(id)).entries());
}

async function submitConcurrent() {
  const button = byId("submit-concurrent");
  button.disabled = true;
  byId("review-status").textContent = "提交中";
  try {
    await Promise.all([
      request("/api/reviews/daily-review/confirmations", { method: "POST", body: JSON.stringify(formValue("operator-a-form")) }),
      request("/api/reviews/daily-review/confirmations", { method: "POST", body: JSON.stringify(formValue("operator-b-form")) }),
    ]);
    await refreshReview();
    byId("review-status").textContent = "提交完成";
  } catch (error) {
    byId("review-status").textContent = error.message;
  } finally {
    button.disabled = false;
  }
}

async function refreshReview() {
  const summary = await request("/api/reviews/daily-review");
  byId("confirmation-list").replaceChildren(...summary.confirmations.map((item) => {
    const node = document.createElement("span");
    node.className = "confirmation";
    node.textContent = `${item.operator} · ${item.content}`;
    return node;
  }));
}

document.querySelectorAll("[data-view]").forEach((button) => {
  button.addEventListener("click", async () => {
    document.querySelectorAll("[data-view]").forEach((item) => item.classList.toggle("is-active", item === button));
    document.querySelectorAll(".view").forEach((item) => item.classList.toggle("is-active", item.id === `view-${button.dataset.view}`));
    if (button.dataset.view === "wrong") await refreshWrongWords();
    if (button.dataset.view === "history") await refreshStats();
    if (button.dataset.view === "review") await refreshReview();
  });
});

document.querySelectorAll("[data-difficulty]").forEach((button) => {
  button.addEventListener("click", async () => {
    state.difficulty = button.dataset.difficulty;
    document.querySelectorAll("[data-difficulty]").forEach((item) => item.classList.toggle("is-active", item === button));
    await loadPrompts();
  });
});

document.querySelectorAll("[data-mode]").forEach((button) => {
  button.addEventListener("click", () => {
    state.mode = button.dataset.mode;
    document.querySelectorAll("[data-mode]").forEach((item) => item.classList.toggle("is-active", item === button));
    renderPrompt();
  });
});

byId("answer-form").addEventListener("submit", submitAnswer);
byId("next-button").addEventListener("click", nextPrompt);
byId("refresh-wrong").addEventListener("click", refreshWrongWords);
byId("submit-concurrent").addEventListener("click", submitConcurrent);

await Promise.all([loadPrompts(), refreshStats()]);
