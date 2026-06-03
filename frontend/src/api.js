const base = "/api";

async function get(path) {
  const res = await fetch(base + path);
  if (!res.ok) throw new Error(`GET ${path} → ${res.status}`);
  return res.json();
}

async function post(path, body) {
  const isFormData = body instanceof FormData;
  const res = await fetch(base + path, {
    method: "POST",
    headers: isFormData
      ? {}
      : { "Content-Type": "application/x-www-form-urlencoded" },
    body: isFormData ? body : new URLSearchParams(body),
  });
  if (!res.ok) throw new Error(`POST ${path} → ${res.status}`);
  return res.status === 204 ? null : res.json();
}

async function del(path) {
  const res = await fetch(base + path, { method: "DELETE" });
  if (!res.ok) throw new Error(`DELETE ${path} → ${res.status}`);
}

export const api = {
  analytics: {
    get: () => get("/analytics"),
  },
  sources: {
    list: () => get("/sources"),
    add: (name) => post("/sources", { name }),
    delete: (id) => del(`/sources/${id}`),
  },
  apps: {
    list: () => get("/apps"),
    get: (id) => get(`/apps/${id}`),
    updateStatus: (id, status, notes) =>
      post(`/apps/${id}/status`, { status, notes: notes ?? "" }),
    updateSource: (id, source) =>
      post(`/apps/${id}/source`, { source }),
    pdfs: (id) => get(`/apps/${id}/pdfs`),
    exportPDF: (id, docType, theme) =>
      post(`/apps/${id}/pdf`, { doc_type: docType, theme: theme ?? '' }),
  },
  cv: {
    list: () => get("/cv"),
    usages: (name) => get(`/cv/${encodeURIComponent(name)}/usages`),
  },
  cl: {
    list: () => get("/cl"),
  },
  themes: {
    list: () => get("/themes"),
    upload: (file) => {
      const fd = new FormData();
      fd.append("theme", file);
      return post("/themes/upload", fd);
    },
  },
  stats: {
    get: () => get("/stats"),
  },
  gotenberg: {
    status: () => get("/gotenberg/status"),
  },
  export: (filePath, theme, method = "auto") =>
    fetch("/api/export", {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams({ file_path: filePath, theme, method }),
    }),
};
