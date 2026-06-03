<template>
  <div class="flex min-h-screen bg-gray-50 text-gray-900">
    <nav class="w-52 bg-gray-900 text-white flex flex-col gap-1 p-4 fixed h-full shrink-0">
      <div class="text-lg font-bold mb-5 px-2">🌶️ Spice Bag</div>
      <RouterLink to="/"        class="px-2 py-1.5 rounded hover:bg-gray-700 text-sm">Overview</RouterLink>
      <RouterLink to="/apps"    class="px-2 py-1.5 rounded hover:bg-gray-700 text-sm">Applications</RouterLink>
      <RouterLink to="/cv"      class="px-2 py-1.5 rounded hover:bg-gray-700 text-sm">Curriculum Vitae</RouterLink>
      <RouterLink to="/cl"      class="px-2 py-1.5 rounded hover:bg-gray-700 text-sm">Cover Letters</RouterLink>
      <RouterLink to="/stats"   class="px-2 py-1.5 rounded hover:bg-gray-700 text-sm">Experience</RouterLink>
      <RouterLink to="/themes"  class="px-2 py-1.5 rounded hover:bg-gray-700 text-sm">Themes</RouterLink>
      <div class="mt-auto pt-4 border-t border-gray-700">
        <RouterLink to="/sources" class="px-2 py-1.5 rounded hover:bg-gray-700 text-sm text-gray-400">Sources</RouterLink>
      </div>
    </nav>
    <main class="ml-52 flex-1 p-8 min-w-0">
      <RouterView />
    </main>
  </div>
</template>
