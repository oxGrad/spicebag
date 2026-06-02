const base = '/api'

async function get(path) {
  const res = await fetch(base + path)
  if (!res.ok) throw new Error(`GET ${path} → ${res.status}`)
  return res.json()
}

async function post(path, body) {
  const isFormData = body instanceof FormData
  const res = await fetch(base + path, {
    method: 'POST',
    headers: isFormData ? {} : { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: isFormData ? body : new URLSearchParams(body),
  })
  if (!res.ok) throw new Error(`POST ${path} → ${res.status}`)
  return res.json()
}

export const api = {
  apps: {
    list: ()              => get('/apps'),
    get:  (id)            => get(`/apps/${id}`),
    updateStatus: (id, status, notes) =>
      post(`/apps/${id}/status`, { status, notes: notes ?? '' }),
  },
  cv: {
    list: () => get('/cv'),
  },
  cl: {
    list: () => get('/cl'),
  },
  themes: {
    list:   ()     => get('/themes'),
    upload: (file) => {
      const fd = new FormData()
      fd.append('theme', file)
      return post('/themes/upload', fd)
    },
  },
  stats: {
    get: () => get('/stats'),
  },
  gotenberg: {
    status: ()  => get('/gotenberg/status'),
    start:  ()  => post('/gotenberg/start', {}),
    stop:   ()  => post('/gotenberg/stop', {}),
  },
  export: (filePath, theme) =>
    fetch('/api/export', {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: new URLSearchParams({ file_path: filePath, theme }),
    }),
}
