const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:8080'

async function api(path, options = {}) {
  const url = `${API_BASE}${path}`
  let res
  try {
    res = await fetch(url, {
      headers: { 'Content-Type': 'application/json', ...options.headers },
      ...options,
    })
  } catch {
    return { ok: false, status: 0, error: 'Network error. Is the server running?' }
  }

  if (!res.ok) {
    let error = `HTTP ${res.status}`
    try {
      const body = await res.json()
      if (body.error) error = body.error
    } catch {
      /* ignore parse errors */
    }
    return { ok: false, status: res.status, error }
  }

  const data = await res.json()
  return { ok: true, status: res.status, data }
}

export function getTodayQuiz() {
  return api('/quiz/today')
}

export function submitAnswer(questionId, option) {
  return api('/quiz/answer', {
    method: 'POST',
    body: JSON.stringify({ question_id: questionId, option }),
  })
}

export function getStats() {
  return api('/stats')
}

export function getTopics() {
  return api('/topics')
}

export function addTopic(name) {
  return api('/topics', {
    method: 'POST',
    body: JSON.stringify({ name }),
  })
}
