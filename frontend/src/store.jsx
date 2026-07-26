import { createContext, useContext, useEffect, useState, useCallback } from 'react'
import { api, setToken, setCompanyId, setUnauthorizedHandler, getToken } from './api.js'

const AuthCtx = createContext(null)
export const useAuth = () => useContext(AuthCtx)

export function AuthProvider({ children }) {
  const [ready, setReady] = useState(false)
  const [user, setUser] = useState(null)
  const [isSuper, setIsSuper] = useState(false)
  const [isTenantAdmin, setIsTenantAdmin] = useState(false)
  const [tenant, setTenant] = useState(null)
  const [companies, setCompanies] = useState([])
  const [company, setCompanyState] = useState(null) // active company view {id,name,role,modules}

  const applyMe = useCallback((data) => {
    setUser(data.user || null)
    setIsSuper(!!data.is_superadmin)
    setIsTenantAdmin(!!data.is_tenant_admin)
    setTenant(data.tenant || null)
    setCompanies(data.companies || [])
    return data.companies || []
  }, [])

  const chooseCompany = useCallback((cmp) => {
    setCompanyState(cmp)
    setCompanyId(cmp ? cmp.id : null)
    if (cmp) localStorage.setItem('oawo_company', String(cmp.id))
  }, [])

  const logout = useCallback(() => {
    setToken('')
    setCompanyId(null)
    localStorage.removeItem('oawo_company')
    setUser(null); setIsSuper(false); setIsTenantAdmin(false); setTenant(null)
    setCompanies([]); setCompanyState(null)
  }, [])

  useEffect(() => { setUnauthorizedHandler(logout) }, [logout])

  // Restore session on load.
  useEffect(() => {
    let cancelled = false
    async function boot() {
      if (!getToken()) { setReady(true); return }
      try {
        const data = await api.get('/auth/me')
        if (cancelled) return
        const cmps = applyMe(data)
        restoreCompany(cmps)
      } catch (e) { logout() }
      if (!cancelled) setReady(true)
    }
    boot()
    return () => { cancelled = true }
  }, []) // eslint-disable-line

  function restoreCompany(cmps) {
    const savedId = Number(localStorage.getItem('oawo_company'))
    const match = cmps.find((c) => c.id === savedId)
    if (match) chooseCompany(match)
    else if (cmps.length === 1) chooseCompany(cmps[0])
    else chooseCompany(null)
  }

  const login = useCallback(async (email, password) => {
    const data = await api.post('/auth/login', { email, password })
    setToken(data.token)
    const cmps = applyMe(data)
    restoreCompany(cmps)
    return data
  }, [applyMe])

  const refreshCompanies = useCallback(async () => {
    const cmps = await api.get('/companies')
    setCompanies(cmps || [])
    // keep active company fresh
    if (company) {
      const m = (cmps || []).find((c) => c.id === company.id)
      if (m) chooseCompany(m)
    }
    return cmps || []
  }, [company, chooseCompany])

  const enabledModules = company ? company.modules : (tenant ? tenant.modules : null)

  const value = {
    ready, user, isSuper, isTenantAdmin, tenant, companies, company,
    enabledModules,
    login, logout, chooseCompany, refreshCompanies, applyMe,
    canWrite: () => company && company.role !== 'viewer',
    canManage: () => isSuper || isTenantAdmin || (company && (company.role === 'owner' || company.role === 'admin')),
    canCreateCompany: () => isSuper || isTenantAdmin,
  }
  return <AuthCtx.Provider value={value}>{children}</AuthCtx.Provider>
}
