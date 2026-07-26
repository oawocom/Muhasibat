import { useState } from 'react'
import { useAuth } from '../store.jsx'
import { useToast, Btn, Logo } from '../ui.jsx'

export default function Login() {
  const auth = useAuth()
  const toast = useToast()
  const [email, setEmail] = useState('admin@oawo.com')
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)

  async function submit(e) {
    e.preventDefault()
    setBusy(true)
    try {
      await auth.login(email, password)
      toast.ok('Xoş gəlmisiniz!')
    } catch (err) {
      toast.err(err.message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="grid h-full place-items-center px-4">
      <form onSubmit={submit} className="w-full max-w-sm rounded-3xl border border-line bg-surface p-9 shadow-2xl">
        <div className="mb-1 flex items-center gap-3">
          <Logo className="h-11 w-11" />
          <h1 className="text-xl font-bold">OAWO Mühasibat</h1>
        </div>
        <p className="mb-6 text-muted">Peşəkar mühasibatlıq sistemi</p>
        <label className="label">Email</label>
        <input className="input mb-3.5" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
        <label className="label">Şifrə</label>
        <input className="input mb-5" type="password" value={password} onChange={(e) => setPassword(e.target.value)} required />
        <Btn variant="primary" className="w-full" disabled={busy}>{busy ? 'Gözləyin...' : 'Daxil ol'}</Btn>
      </form>
    </div>
  )
}
