import { useEffect, useMemo, useState } from 'react'
import {
  addTopic,
  getStats,
  getTodayQuiz,
  getTopics,
  submitAnswer,
} from './api.js'
import './App.css'

const TABS = {
  QUIZ: '#/quiz',
  STATS: '#/stats',
  TOPICS: '#/topics',
}

function useHash() {
  const [hash, setHash] = useState(() => window.location.hash || TABS.QUIZ)
  useEffect(() => {
    const onHashChange = () => setHash(window.location.hash || TABS.QUIZ)
    window.addEventListener('hashchange', onHashChange)
    return () => window.removeEventListener('hashchange', onHashChange)
  }, [])
  return hash
}

function Loading() {
  return <div className="loading">Loading…</div>
}

function Error({ message, onRetry }) {
  return (
    <div className="error" role="alert">
      <p>{message || 'Something went wrong.'}</p>
      {onRetry && (
        <button type="button" className="retry-btn" onClick={onRetry}>
          Retry
        </button>
      )}
    </div>
  )
}

function Empty({ title, message }) {
  return (
    <div className="empty">
      <h2>{title}</h2>
      <p>{message}</p>
    </div>
  )
}

function isToday(dateString) {
  const today = new Date().toLocaleDateString('en-CA')
  return dateString === today
}

function QuizView() {
  const [quiz, setQuiz] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [index, setIndex] = useState(0)
  const [feedback, setFeedback] = useState(null)
  const [answered, setAnswered] = useState(new Set())
  const [done, setDone] = useState(false)

  const load = async () => {
    setLoading(true)
    setError(null)
    const result = await getTodayQuiz()
    if (!result.ok) {
      setError(
        result.status === 404
          ? 'No quiz available today. Check back later!'
          : result.error || 'Could not load today\'s quiz.'
      )
      setQuiz(null)
    } else {
      setQuiz(result.data)
      setIndex(0)
      setFeedback(null)
      setAnswered(new Set())
      setDone(false)
    }
    setLoading(false)
  }

  useEffect(() => {
    load()
  }, [])

  const handleAnswer = async (option) => {
    const question = quiz.questions[index]
    if (answered.has(question.id)) return

    const result = await submitAnswer(question.id, option)
    if (!result.ok) {
      setError(result.error || 'Could not submit answer.')
      return
    }

    setAnswered((prev) => new Set(prev).add(question.id))
    setFeedback({ option, ...result.data })
  }

  const handleNext = () => {
    if (index + 1 >= quiz.questions.length) {
      setDone(true)
    } else {
      setIndex((i) => i + 1)
      setFeedback(null)
    }
  }

  if (loading) return <Loading />
  if (error && !quiz) return <Error message={error} onRetry={load} />
  if (!quiz || quiz.questions.length === 0) {
    return <Empty title="No quiz today" message="There are no questions available right now." />
  }

  if (done) {
    return (
      <div className="done-state">
        <h2>All done!</h2>
        <p>You answered all {quiz.questions.length} questions.</p>
        <a href={TABS.STATS} className="btn done-btn">
          View stats
        </a>
      </div>
    )
  }

  const question = quiz.questions[index]
  const isAnswered = answered.has(question.id)

  return (
    <section aria-label="Today’s quiz">
      {!isToday(quiz.batch_date) && (
        <div className="fallback-note">
          Showing a previous quiz from {quiz.batch_date}.
        </div>
      )}
      <div className="quiz-meta">
        <span>
          Question {index + 1} of {quiz.questions.length}
        </span>
        <span>{quiz.batch_date}</span>
      </div>
      <div className="question-card">
        <span className="question-topic">{question.topic}</span>
        <h2 className="question-prompt">{question.prompt}</h2>
        <div className="options" role="group" aria-label="Answer options">
          {question.options.map((opt) => {
            let stateClass = ''
            if (isAnswered && feedback) {
              if (opt.id === feedback.option) {
                stateClass = feedback.correct ? 'correct' : 'wrong'
              }
            }
            return (
              <button
                key={opt.id}
                type="button"
                className={`option-btn ${stateClass}`}
                onClick={() => handleAnswer(opt.id)}
                disabled={isAnswered}
                aria-pressed={isAnswered && feedback?.option === opt.id}
              >
                <span className="option-key" aria-hidden="true">
                  {opt.id}
                </span>
                <span>{opt.text}</span>
              </button>
            )
          })}
        </div>
        {feedback && (
          <div
            className={`feedback ${feedback.correct ? 'correct' : 'wrong'}`}
            role="status"
            aria-live="polite"
          >
            <h3>{feedback.correct ? 'Correct!' : 'Not quite.'}</h3>
            {feedback.explanation && <p>{feedback.explanation}</p>}
          </div>
        )}
        {isAnswered && (
          <button type="button" className="next-btn" onClick={handleNext}>
            {index + 1 >= quiz.questions.length ? 'Finish' : 'Next question'}
          </button>
        )}
      </div>
      {error && <Error message={error} onRetry={() => setError(null)} />}
    </section>
  )
}

function StatsView() {
  const [stats, setStats] = useState(null)
  const [topics, setTopics] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  const load = async () => {
    setLoading(true)
    setError(null)
    const [statsResult, topicsResult] = await Promise.all([getStats(), getTopics()])
    if (!statsResult.ok || !topicsResult.ok) {
      setError(statsResult.error || topicsResult.error || 'Could not load stats.')
      setLoading(false)
      return
    }
    setStats(statsResult.data)
    setTopics(topicsResult.data)
    setLoading(false)
  }

  useEffect(() => {
    load()
  }, [])

  const joined = useMemo(() => {
    if (!stats || !topics) return []
    const byId = new Map(topics.map((t) => [t.id, t]))
    return stats.map((s) => ({
      ...s,
      name: byId.get(s.topic_id)?.name || `Topic ${s.topic_id}`,
    }))
  }, [stats, topics])

  const totals = useMemo(() => {
    const correct = joined.reduce((sum, s) => sum + s.correct_count, 0)
    const wrong = joined.reduce((sum, s) => sum + s.wrong_count, 0)
    return { correct, wrong, total: correct + wrong }
  }, [joined])

  if (loading) return <Loading />
  if (error) return <Error message={error} onRetry={load} />

  return (
    <section aria-label="Stats">
      <h1 style={{ marginBottom: 20 }}>Your stats</h1>
      <div className="stats-summary">
        <div className="summary-card">
          <span className="value">{totals.total}</span>
          <span className="label">Answered</span>
        </div>
        <div className="summary-card">
          <span className="value">{totals.correct}</span>
          <span className="label">Correct</span>
        </div>
        <div className="summary-card">
          <span className="value">{totals.wrong}</span>
          <span className="label">Wrong</span>
        </div>
      </div>
      <div className="stats-list">
        {joined.length === 0 ? (
          <div className="stat-row empty">
            No stats yet. Answer some questions to see them here.
          </div>
        ) : (
          joined.map((s) => (
            <div key={s.topic_id} className="stat-row">
              <div className="stat-header">
                <span className="stat-name">{s.name}</span>
                <span className="stat-accuracy">
                  {Math.round(s.accuracy_rate)}%
                </span>
              </div>
              <div className="stat-bar" aria-hidden="true">
                <div
                  className="stat-bar-fill"
                  style={{ width: `${s.accuracy_rate}%` }}
                />
              </div>
              <div className="stat-footer">
                <span className="correct">{s.correct_count} correct</span>
                <span className="wrong">{s.wrong_count} wrong</span>
                <span>
                  Last practiced:{' '}
                  {s.last_practiced_at
                    ? new Date(s.last_practiced_at).toLocaleDateString()
                    : 'never'}
                </span>
              </div>
            </div>
          ))
        )}
      </div>
    </section>
  )
}

function TopicsView() {
  const [topics, setTopics] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [name, setName] = useState('')
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState(null)

  const load = async () => {
    setLoading(true)
    setError(null)
    const result = await getTopics()
    if (!result.ok) {
      setError(result.error || 'Could not load topics.')
    } else {
      setTopics(result.data)
    }
    setLoading(false)
  }

  useEffect(() => {
    load()
  }, [])

  const handleSubmit = async (e) => {
    e.preventDefault()
    const trimmed = name.trim()
    if (!trimmed) return
    setSaving(true)
    setSaveError(null)
    const result = await addTopic(trimmed)
    if (!result.ok) {
      if (result.status === 409) {
        setSaveError('Topic already exists.')
      } else if (result.status === 400) {
        setSaveError('Name is required.')
      } else {
        setSaveError(result.error || 'Could not add topic.')
      }
    } else {
      setName('')
      await load()
    }
    setSaving(false)
  }

  if (loading) return <Loading />
  if (error) return <Error message={error} onRetry={load} />

  return (
    <section aria-label="Topics">
      <h1 style={{ marginBottom: 20 }}>Topics</h1>
      <form className="topics-form" onSubmit={handleSubmit}>
        <input
          type="text"
          placeholder="New topic name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          aria-label="New topic name"
        />
        <button type="submit" disabled={saving || !name.trim()}>
          Add topic
        </button>
      </form>
      {saveError && <p className="inline-error" role="alert">{saveError}</p>}
      {topics.length === 0 ? (
        <Empty title="No topics" message="Add a topic to get started." />
      ) : (
        <ul className="topics-list">
          {topics.map((t) => (
            <li key={t.id}>
              <span className="topic-name">{t.name}</span>
              <span className="topic-weight">weight {t.weight}</span>
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}

function App() {
  const hash = useHash()

  return (
    <>
      <header className="app-header">
        <a className="brand" href={TABS.QUIZ}>
          Quizme
        </a>
        <nav className="nav" aria-label="Main">
          <a href={TABS.QUIZ} className={hash === TABS.QUIZ ? 'active' : ''}>
            Quiz
          </a>
          <a href={TABS.STATS} className={hash === TABS.STATS ? 'active' : ''}>
            Stats
          </a>
          <a href={TABS.TOPICS} className={hash === TABS.TOPICS ? 'active' : ''}>
            Topics
          </a>
        </nav>
      </header>
      <main>
        {hash === TABS.STATS && <StatsView />}
        {hash === TABS.TOPICS && <TopicsView />}
        {(hash === TABS.QUIZ || !Object.values(TABS).includes(hash)) && <QuizView />}
      </main>
    </>
  )
}

export default App
