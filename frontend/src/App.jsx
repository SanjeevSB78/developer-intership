import { useEffect, useMemo, useState } from 'react'
import { ArrowLeft, ArrowRight, BarChart3, Check, CircleHelp, LoaderCircle, LockKeyhole, LogOut, Plus, Radio, Share2, Sparkles, X } from 'lucide-react'

const API = (import.meta.env.VITE_API_URL || '').replace(/\/$/, '')
const tokenKey = 'pulse_token'
async function request(path, { method = 'GET', body, token } = {}) {
  const response = await fetch(`${API}/api${path}`, { method, headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) }, ...(body ? { body: JSON.stringify(body) } : {}) })
  const data = await response.json().catch(() => ({}))
  if (!response.ok) throw new Error(data.error || 'Something went wrong. Please try again.')
  return data
}
function voterID(id) { const key = `pulse_voter_${id}`; let value = localStorage.getItem(key); if (!value) { value = crypto.randomUUID(); localStorage.setItem(key, value) }; return value }
function usePoll(id) {
  const [poll, setPoll] = useState(null), [error, setError] = useState(''), [live, setLive] = useState(false)
  useEffect(() => { if (!id) return; let source; let stopped = false
    request(`/polls/${id}`).then(data => {
      if (stopped) return
      setPoll(data)
      source = new EventSource(`${API}/api/polls/${id}/events`)
      source.addEventListener('open', () => setLive(true))
      source.addEventListener('error', () => setLive(false))
      const updateResults = event => {
        try {
          const update = JSON.parse(event.data)
          setPoll(old => old ? {
            ...old,
            counts: update.counts || { ...old.counts, [update.optionId]: update.count },
            total: update.total,
          } : old)
        } catch {}
      }
      source.addEventListener('snapshot', updateResults)
      source.addEventListener('vote', updateResults)
    }).catch(e => { if (!stopped) setError(e.message) })
    return () => { stopped = true; source?.close() }
  }, [id])
  return { poll, setPoll, error, live }
}
function App() {
  const pollId = useMemo(() => location.pathname.match(/^\/p\/([a-f\d]{24})$/i)?.[1] || new URLSearchParams(location.search).get('poll'), [])
  const [token, setToken] = useState(() => localStorage.getItem(tokenKey) || ''), [user, setUser] = useState(null)
  const [authOpen, setAuthOpen] = useState(false), [authMode, setAuthMode] = useState('signup'), [notice, setNotice] = useState('')
  const { poll, setPoll, error, live } = usePoll(pollId)
  useEffect(() => { if (!token) return; request('/auth/me', { token }).then(setUser).catch(() => { localStorage.removeItem(tokenKey); setToken('') }) }, [token])
  function flash(text) { setNotice(text); setTimeout(() => setNotice(''), 2800) }
  async function share(id) { const url = `${location.origin}/p/${id}`; try { await navigator.clipboard.writeText(url); flash('Poll link copied to clipboard') } catch { prompt('Copy this poll link', url) } }
  function signOut() { localStorage.removeItem(tokenKey); setToken(''); setUser(null); flash('You’re signed out') }

  return <div className="app-shell">
    <header className="topbar"><a className="brand" href="/" aria-label="Pulse home"><span className="brand-mark"><Radio size={19}/></span><span>pulse<span className="brand-dot">.</span></span></a><nav><a href="#how">How it works</a><a href="#about">About</a></nav><div className="top-actions">{user ? <><span className="user-chip"><span className="avatar">{user.name?.[0]?.toUpperCase()}</span>{user.name}</span><button className="icon-btn" aria-label="Sign out" onClick={signOut}><LogOut size={17}/></button></> : <><button className="text-button" onClick={() => { setAuthMode('login'); setAuthOpen(true) }}>Log in</button><button className="button button-dark top-cta" onClick={() => { setAuthMode('signup'); setAuthOpen(true) }}>Get started <ArrowRight size={15}/></button></>}</div></header>
    <main>
      {pollId ? <PollPage poll={poll} setPoll={setPoll} error={error} live={live} share={share} flash={flash}/> : <>
        <section className="hero"><div className="hero-copy"><div className="eyebrow"><span className="eyebrow-line"/>A little more signal, a lot less noise</div><h1>Good questions<br/>get <em>great answers.</em></h1><p className="hero-intro">Create a poll, share it with your people, and watch opinions roll in. No refresh button required.</p><div className="hero-buttons"><button className="button button-dark button-large" onClick={() => user ? document.getElementById('create')?.scrollIntoView({ behavior: 'smooth' }) : (setAuthMode('signup'), setAuthOpen(true))}>Create your first poll <ArrowRight size={17}/></button><a href="#how" className="learn-link">See how it works <ArrowRight size={15}/></a></div><div className="proof"><div className="proof-avatars"><span>J</span><span>M</span><span>A</span><span>+</span></div><span>For curious minds and good teams</span></div></div><HeroDemo/></section>
        <section className="ticker"><span>ASK BETTER QUESTIONS</span><i/> <span>GET THE ROOM TALKING</span><i/> <span>SEE WHAT PEOPLE THINK</span><i/> <span>NO REFRESH NEEDED</span><i/> <span>ASK BETTER QUESTIONS</span></section>
        <section id="create" className="create-section"><div className="section-head"><div><div className="eyebrow"><span className="eyebrow-line"/>THE GOOD PART</div><h2>Start with a<br/><em>question.</em></h2></div><p>One clear question can change the whole conversation. Make yours count.</p></div>{user ? <CreatePoll token={token} onCreated={id => { location.href = `/p/${id}` }}/> : <div className="locked-card"><span className="lock-icon"><LockKeyhole size={18}/></span><div><h3>Your next great question starts here.</h3><p>Create a free account to build a poll and make it yours.</p></div><button className="button button-dark" onClick={() => { setAuthMode('signup'); setAuthOpen(true) }}>Make an account <ArrowRight size={15}/></button></div>}</section>
        <section id="how" className="steps-section"><div className="steps-top"><div><div className="eyebrow"><span className="eyebrow-line"/>SIMPLE AS ONE, TWO, THREE</div><h2>From wondering<br/>to <em>knowing.</em></h2></div><p>No accounts for your voters. No complicated setup. Just good questions and instant answers.</p></div><div className="step-grid"><Step n="01" icon={<Sparkles size={20}/>} title="Ask away" body="Write your question and add a few choices. Keep it simple, or get wonderfully specific."/><Step n="02" icon={<Share2 size={20}/>} title="Pass it around" body="Copy your unique link and drop it into the group chat, the meeting, or wherever your people are."/><Step n="03" icon={<BarChart3 size={20}/>} title="Watch it happen" body="Votes show up as they come in. You can practically feel the room making up its mind."/></div></section>
        <section id="about" className="quote-section"><div className="quote-stamp"><Radio size={21}/><span>LIVE<br/>BY NATURE</span></div><p>“The best ideas don’t happen in a vacuum. They happen when everyone gets a say.”</p><span className="quote-credit">A small thought from the people at pulse</span></section>
        <footer className="footer"><a className="brand" href="/"><span className="brand-mark"><Radio size={18}/></span><span>pulse<span className="brand-dot">.</span></span></a><span>Built for the joy of finding out.</span><a href="#top" onClick={e => { e.preventDefault(); window.scrollTo({ top: 0, behavior: 'smooth' }) }}>Back to top ↑</a></footer>
      </>}
    </main>
    {authOpen && <AuthModal mode={authMode} setMode={setAuthMode} onClose={() => setAuthOpen(false)} onSuccess={(t,u) => { localStorage.setItem(tokenKey,t); setToken(t); setUser(u); setAuthOpen(false); flash('You’re in. Let’s make something good.') }}/>}
    {notice && <div className="toast"><Check size={16}/>{notice}</div>}
  </div>
}
function HeroDemo() { return <div className="hero-visual"><div className="visual-note note-top"><Sparkles size={13}/> THE GROUP CHAT, SETTLED</div><div className="visual-card"><div className="visual-card-head"><span className="live-tag"><i/> LIVE POLL</span><span className="more-dots">•••</span></div><div className="demo-question">What’s the move<br/>for Friday night?</div><div className="demo-options"><div className="demo-option chosen"><span className="demo-check"><Check size={12}/></span><span>Try that new ramen place</span><span className="demo-pct">48%</span><i style={{width:'48%'}}/></div><div className="demo-option"><span className="demo-empty"/><span>Movie night, no notes</span><span className="demo-pct">32%</span><i style={{width:'32%'}}/></div><div className="demo-option"><span className="demo-empty"/><span>Let’s see where we end up</span><span className="demo-pct">20%</span><i style={{width:'20%'}}/></div></div><div className="demo-foot"><div className="mini-avatars"><b>M</b><b>K</b><b>A</b><b>+</b></div><span>25 people have weighed in</span><span className="pulse-live"><i/> LIVE</span></div></div><div className="visual-note note-bottom"><span className="sparkle-orb"><Sparkles size={17}/></span><span>That was quick.<br/><strong>And kind of fun.</strong></span></div><div className="visual-sun"/></div> }
function Step({ n, icon, title, body }) { return <article className="step-card"><div className="step-card-top"><span className="step-icon">{icon}</span><span className="step-n">{n}</span></div><h3>{title}</h3><p>{body}</p><div className="step-rule"/></article> }
function CreatePoll({ token, onCreated }) {
  const [question,setQuestion]=useState(''),[choices,setChoices]=useState(['','']),[busy,setBusy]=useState(false),[error,setError]=useState('')
  async function submit(e){e.preventDefault();setError('');setBusy(true);try{const p=await request('/polls',{method:'POST',token,body:{question,options:choices}});onCreated(p.id)}catch(e){setError(e.message)}finally{setBusy(false)}}
  return <form className="create-card" onSubmit={submit}><div className="form-label-row"><label htmlFor="question">YOUR QUESTION</label><span>{question.length}/180</span></div><textarea id="question" placeholder="What do you want to know?" maxLength="180" value={question} onChange={e=>setQuestion(e.target.value)} required minLength="8"/><div className="choice-label">ANSWER CHOICES <span>At least two, up to six</span></div><div className="choice-list">{choices.map((v,i)=><div className="choice-input" key={i}><span>{String(i+1).padStart(2,'0')}</span><input aria-label={`Choice ${i+1}`} maxLength="80" required placeholder="Add a choice" value={v} onChange={e=>setChoices(choices.map((x,j)=>j===i?e.target.value:x))}/>{choices.length>2&&<button type="button" aria-label="Remove choice" onClick={()=>setChoices(choices.filter((_,j)=>j!==i))}><X size={16}/></button>}</div>)}</div>{choices.length<6&&<button type="button" className="add-choice" onClick={()=>setChoices([...choices,''])}><Plus size={14}/> Add another choice</button>}{error&&<p className="form-error">{error}</p>}<div className="create-submit"><span><LockKeyhole size={13}/> Poll creation requires your account</span><button className="button button-dark" disabled={busy}>{busy?<LoaderCircle className="spin" size={16}/>:<>Create poll <ArrowRight size={15}/></>}</button></div></form>
}
function PollPage({poll,setPoll,error,live,share,flash}) {
  const [selected,setSelected]=useState(''),[busy,setBusy]=useState(false),[voteError,setVoteError]=useState(''),[voted,setVoted]=useState(false)
  async function vote(){if(!selected)return;setBusy(true);setVoteError('');try{const result=await request(`/polls/${poll.id}/votes`,{method:'POST',body:{optionId:selected,voterId:voterID(poll.id)}});setPoll(p=>({...p,counts:{...p.counts,[selected]:result.count},total:result.total}));setVoted(true)}catch(e){setVoteError(e.message);if(e.message.includes('already voted'))setVoted(true)}finally{setBusy(false)}}
  if(error)return <div className="poll-error"><span className="error-mark">?</span><h1>We can’t find that poll.</h1><p>{error}</p><a className="button button-dark" href="/">Head home <ArrowRight size={15}/></a></div>
  if(!poll)return <div className="poll-loading"><LoaderCircle className="spin"/> Getting the question ready…</div>
  return <div className="poll-page"><a className="back-link" href="/"><ArrowLeft size={15}/> Back to pulse</a><div className="poll-heading"><div className="eyebrow"><span className="eyebrow-line"/>A QUESTION FOR THE ROOM</div><h1>{poll.question}</h1><p>Choose the answer that feels right. Your vote is anonymous.</p></div><div className="poll-card"><div className="poll-card-head"><span className={`live-tag ${live?'':'offline'}`}><i/>{live?'LIVE RESULTS':'CONNECTING'}</span><span>{poll.total} {poll.total===1?'vote':'votes'}</span></div><div className="poll-options">{poll.options.map(o=>{const n=poll.counts?.[o.id]||0,pct=poll.total?Math.round(n*100/poll.total):0;return <button key={o.id} className={`poll-option ${selected===o.id?'selected':''} ${voted?'results':''}`} onClick={()=>!voted&&setSelected(o.id)} disabled={voted}><span className="radio-dot">{selected===o.id&&<i/>}</span><span className="option-text">{o.text}</span>{voted&&<><span className="option-pct">{pct}%</span><i className="result-bar" style={{width:`${pct}%`}}/></>}</button>})}</div>{voteError&&<p className="form-error">{voteError}</p>}{!voted?<div className="poll-card-actions"><span>One vote per browser</span><button className="button button-dark" disabled={!selected||busy} onClick={vote}>{busy?<LoaderCircle className="spin" size={15}/>:<>Cast your vote <ArrowRight size={15}/></>}</button></div>:<div className="voted-message"><span className="success-icon"><Check size={15}/></span><span>Your vote is in. Watch the room weigh in.</span><span className="live-small"><i/> updating live</span></div>}</div><button className="share-poll" onClick={()=>share(poll.id)}><Share2 size={15}/> Share this poll <span>Invite someone to weigh in <ArrowRight size={13}/></span></button><div className="poll-footnote"><CircleHelp size={15}/> Results update automatically while this page is open.</div></div>
}
function AuthModal({mode,setMode,onClose,onSuccess}) {
  const [name,setName]=useState(''),[email,setEmail]=useState(''),[password,setPassword]=useState(''),[error,setError]=useState(''),[busy,setBusy]=useState(false)
  async function submit(e){e.preventDefault();setError('');setBusy(true);try{const data=await request(`/auth/${mode}`,{method:'POST',body:{...(mode==='signup'?{name}:{}),email,password}});onSuccess(data.token,data.user)}catch(e){setError(e.message)}finally{setBusy(false)}}
  return <div className="modal-backdrop" onMouseDown={e=>e.target===e.currentTarget&&onClose()}><section className="auth-modal" role="dialog" aria-modal="true" aria-labelledby="auth-title"><button className="modal-close" onClick={onClose} aria-label="Close"><X size={18}/></button><div className="auth-mark"><Radio size={21}/></div><div className="eyebrow"><span className="eyebrow-line"/>A GOOD PLACE TO START</div><h2 id="auth-title">{mode==='signup'?<>Let’s get<br/><em>curious.</em></>:<>Welcome<br/><em>back.</em></>}</h2><p>{mode==='signup'?'Make an account to create and manage your polls.':'Sign in to pick up where you left off.'}</p><form onSubmit={submit}>{mode==='signup'&&<label>Your name<input autoComplete="name" minLength="2" maxLength="60" required value={name} onChange={e=>setName(e.target.value)} placeholder="How should we call you?"/></label>}<label>Email address<input type="email" autoComplete="email" required value={email} onChange={e=>setEmail(e.target.value)} placeholder="you@example.com"/></label><label>Password<input type="password" autoComplete={mode==='signup'?'new-password':'current-password'} minLength="10" maxLength="72" required value={password} onChange={e=>setPassword(e.target.value)} placeholder={mode==='signup'?'At least 10 characters':'Your password'}/></label>{error&&<p className="form-error">{error}</p>}<button className="button button-dark auth-submit" disabled={busy}>{busy?<LoaderCircle className="spin" size={16}/>:<>{mode==='signup'?'Create my account':'Log in'} <ArrowRight size={15}/></>}</button></form><div className="auth-switch">{mode==='signup'?'Already have an account?':'New around here?'} <button onClick={()=>{setMode(mode==='signup'?'login':'signup');setError('')}}>{mode==='signup'?'Log in':'Make an account'}</button></div><div className="auth-fine"><LockKeyhole size={12}/> Your email stays yours.</div></section></div>
}

export default App
