import { useState, useEffect, useCallback } from 'react'
import { api } from './api.js'

// useList fetches a collection/endpoint and exposes a reload().
export function useList(path, deps = []) {
  const [data, setData] = useState(null)
  const [error, setError] = useState(null)
  const reload = useCallback(() => {
    setError(null)
    api.get(path).then(setData).catch((e) => setError(e.message))
  }, [path])
  useEffect(() => { reload() }, deps) // eslint-disable-line
  return { data, error, reload, setData }
}
