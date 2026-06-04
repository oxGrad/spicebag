import { createRouter, createWebHistory } from "vue-router";

const routes = [
  { path: "/",         component: () => import("../views/HomeView.vue") },
  { path: "/apps",     component: () => import("../views/AppsView.vue") },
  { path: "/apps/:id", component: () => import("../views/AppDetailView.vue") },
  { path: "/cv",       component: () => import("../views/CVListView.vue") },
  { path: "/cv/:name", component: () => import("../views/CVView.vue") },
  { path: "/cl",       component: () => import("../views/CLListView.vue") },
  { path: "/cl/:name", component: () => import("../views/CLView.vue") },
  { path: "/themes",   component: () => import("../views/ThemesView.vue") },
  { path: "/sources",  component: () => import("../views/SourcesView.vue") },
  { path: "/stats",    component: () => import("../views/StatsView.vue") },
  { path: "/memory",   component: () => import("../views/MemoryView.vue") },
];

export default createRouter({ history: createWebHistory(), routes });
