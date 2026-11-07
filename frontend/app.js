/**
 * FilmLog — Frontend Companion for GDGoC Backend Assignment
 * Lightweight, zero-build Single Page Application.
 */

// ==========================================
// 1. STATE MANAGEMENT
// ==========================================
const state = {
  apiBaseUrl: localStorage.getItem('filmlog_api_url') || 'http://localhost:8000/api',
  token: localStorage.getItem('filmlog_token') || null,
  user: JSON.parse(localStorage.getItem('filmlog_user') || 'null'),
  currentTab: 'discover',
  authMode: 'login', // 'login' | 'register'
  allMovies: [],
  filteredMovies: [],
  watchlist: [], // Array of movie objects or IDs
  watchedList: [], // Array of watched review items
  selectedGenre: 'ALL',
  searchQuery: '',
  activeDetailMovie: null,
  logs: []
};

// ==========================================
// 2. HTTP CLIENT & NETWORK INSPECTOR
// ==========================================
async function apiFetch(endpoint, options = {}) {
  const url = `${state.apiBaseUrl.replace(/\/$/, '')}/${endpoint.replace(/^\//, '')}`;
  const method = options.method || 'GET';
  const headers = {
    'Content-Type': 'application/json',
    ...(options.headers || {})
  };

  if (state.token) {
    headers['Authorization'] = `Bearer ${state.token}`;
  }

  const startTime = performance.now();
  let status = 0;
  let responseData = null;
  let ok = false;

  try {
    const res = await fetch(url, {
      ...options,
      headers
    });
    status = res.status;
    ok = res.ok;

    const contentType = res.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
      responseData = await res.json();
    } else {
      responseData = await res.text();
    }

    logApiRequest({
      time: new Date().toLocaleTimeString(),
      method,
      url,
      status,
      ok,
      duration: Math.round(performance.now() - startTime)
    });

    if (!ok) {
      const errorMsg = (typeof responseData === 'object' && (responseData.message || responseData.error))
        ? (responseData.message || responseData.error)
        : `Request failed with status ${status}`;
      throw new Error(errorMsg);
    }

    return responseData;
  } catch (err) {
    if (status === 0) {
      logApiRequest({
        time: new Date().toLocaleTimeString(),
        method,
        url,
        status: 'CORS/ERR',
        ok: false,
        duration: Math.round(performance.now() - startTime)
      });
    }
    throw err;
  }
}

function logApiRequest(logItem) {
  state.logs.unshift(logItem);
  if (state.logs.length > 20) state.logs.pop();

  const container = document.getElementById('logs-content');
  if (!container) return;

  container.innerHTML = state.logs.map(l => `
    <div class="flex items-center justify-between py-1 border-b border-brand-border/40">
      <div class="flex items-center gap-2 overflow-hidden">
        <span class="text-[10px] px-1.5 py-0.5 rounded font-bold ${
          l.ok ? 'bg-emerald-950 text-emerald-400 border border-emerald-800' : 'bg-rose-950 text-rose-400 border border-rose-800'
        }">${l.method}</span>
        <span class="text-white truncate max-w-xs sm:max-w-md">${l.url}</span>
      </div>
      <div class="flex items-center gap-3 shrink-0">
        <span class="${l.ok ? 'text-emerald-400' : 'text-rose-400'} font-bold">${l.status}</span>
        <span class="text-brand-muted text-[10px]">${l.duration}ms</span>
        <span class="text-brand-muted text-[10px] hidden sm:inline">${l.time}</span>
      </div>
    </div>
  `).join('');
}

function toggleLogsDrawer() {
  const content = document.getElementById('logs-content');
  const icon = document.getElementById('log-drawer-icon');
  const text = document.getElementById('log-drawer-text');
  const isHidden = content.classList.contains('hidden');

  if (isHidden) {
    content.classList.remove('hidden');
    icon.style.transform = 'rotate(180deg)';
    text.textContent = 'Hide Logs';
  } else {
    content.classList.add('hidden');
    icon.style.transform = 'rotate(0deg)';
    text.textContent = 'Show Logs';
  }
}

// ==========================================
// 3. INITIALIZATION
// ==========================================
document.addEventListener('DOMContentLoaded', () => {
  lucide.createIcons();
  updateApiUrlDisplay();
  renderAuthUI();
  loadDiscoverMovies();
  if (state.token) {
    loadWatchlist();
    loadWatchedList();
  }
});

function updateApiUrlDisplay() {
  const display = document.getElementById('api-url-display');
  const input = document.getElementById('api-base-url-input');
  if (display) display.textContent = state.apiBaseUrl;
  if (input) input.value = state.apiBaseUrl;
}

// ==========================================
// 4. TAB NAVIGATION
// ==========================================
function switchTab(tabId) {
  state.currentTab = tabId;

  // Update tabs UI
  document.querySelectorAll('.tab-content').forEach(el => el.classList.add('hidden'));
  const activeSection = document.getElementById(`tab-${tabId}`);
  if (activeSection) activeSection.classList.remove('hidden');

  // Update desktop nav buttons
  document.querySelectorAll('.nav-tab').forEach(btn => {
    btn.classList.remove('text-white', 'bg-brand-card');
    btn.classList.add('text-brand-muted');
  });
  const activeNav = document.getElementById(`nav-${tabId}`);
  if (activeNav) {
    activeNav.classList.remove('text-brand-muted');
    activeNav.classList.add('text-white', 'bg-brand-card');
  }

  // Refresh tab data if needed
  if (tabId === 'watchlist') {
    if (!state.token) {
      showToast('Silakan login terlebih dahulu untuk melihat Watchlist', 'info');
      openAuthModal('login');
      return;
    }
    loadWatchlist();
  } else if (tabId === 'watched') {
    if (!state.token) {
      showToast('Silakan login terlebih dahulu untuk melihat Watched list', 'info');
      openAuthModal('login');
      return;
    }
    loadWatchedList();
  }

  lucide.createIcons();
}

// ==========================================
// 5. MOVIES (DISCOVER)
// ==========================================
async function loadDiscoverMovies(shuffle = false) {
  const statusEl = document.getElementById('movie-grid-status');
  const gridEl = document.getElementById('movie-grid');
  statusEl.classList.remove('hidden');

  try {
    // Try fetching from applicant's backend: GET /api/movies
    const data = await apiFetch('/movies');
    // Normalize response: could be array or { data: [...] }
    state.allMovies = Array.isArray(data) ? data : (data.data || data.movies || []);
    
    if (shuffle) {
      state.allMovies.sort(() => Math.random() - 0.5);
      showToast('Daftar film berhasil diacak (shuffle)!', 'success');
    }

    applyMovieFilters();
    statusEl.classList.add('hidden');
    document.getElementById('api-status-dot').className = 'w-2 h-2 rounded-full bg-emerald-400';
  } catch (err) {
    statusEl.classList.add('hidden');
    document.getElementById('api-status-dot').className = 'w-2 h-2 rounded-full bg-rose-500';

    // Fallback: If backend is not running, load local static seed so applicant can see UI layout
    console.warn('Backend not reachable, loading fallback local seed:', err.message);
    try {
      const fallbackRes = await fetch('../data/movies.json');
      state.allMovies = await fallbackRes.json();
      if (shuffle) state.allMovies.sort(() => Math.random() - 0.5);
      applyMovieFilters();
      showToast('⚠️ Backend belum aktif. Menampilkan data lokal (fallback). Pastikan server backend Anda berjalan di ' + state.apiBaseUrl, 'warning');
    } catch (fallbackErr) {
      gridEl.innerHTML = `
        <div class="col-span-full py-12 text-center text-rose-400 bg-rose-950/20 border border-rose-900/40 rounded-xl p-6">
          <p class="font-bold mb-1">Gagal terhubung ke backend (${state.apiBaseUrl})</p>
          <p class="text-xs text-brand-muted">Pastikan server backend Anda menyala dan mengizinkan CORS.</p>
        </div>
      `;
    }
  }
}

function handleMovieFilter() {
  state.searchQuery = document.getElementById('movie-search').value.toLowerCase().trim();
  applyMovieFilters();
}

function filterByGenre(genre) {
  state.selectedGenre = genre;
  document.querySelectorAll('.genre-btn').forEach(btn => {
    if (btn.innerText.trim() === genre || (genre === 'ALL' && btn.innerText.includes('All'))) {
      btn.className = 'genre-btn px-3 py-1.5 rounded-full bg-white text-black font-semibold';
    } else {
      btn.className = 'genre-btn px-3 py-1.5 rounded-full bg-brand-surface border border-brand-border text-brand-muted hover:text-white';
    }
  });
  applyMovieFilters();
}

function applyMovieFilters() {
  state.filteredMovies = state.allMovies.filter(movie => {
    const matchesSearch = !state.searchQuery || 
      movie.title.toLowerCase().includes(state.searchQuery) ||
      (movie.director && movie.director.toLowerCase().includes(state.searchQuery));

    let movieGenres = Array.isArray(movie.genres) ? movie.genres : (movie.genres ? movie.genres.split(',') : []);
    movieGenres = movieGenres.map(g => g.trim());

    const matchesGenre = state.selectedGenre === 'ALL' || movieGenres.includes(state.selectedGenre);

    return matchesSearch && matchesGenre;
  });

  renderMovieGrid();
}

function renderMovieGrid() {
  const gridEl = document.getElementById('movie-grid');
  if (!gridEl) return;

  if (state.filteredMovies.length === 0) {
    gridEl.innerHTML = `
      <div class="col-span-full py-16 text-center text-brand-muted">
        <p class="text-base font-semibold text-white">Tidak ada film yang cocok</p>
        <p class="text-xs mt-1">Coba ubah kata kunci pencarian atau filter genre.</p>
      </div>
    `;
    return;
  }

  gridEl.innerHTML = state.filteredMovies.map(movie => {
    const isWatchlisted = isMovieInWatchlist(movie.id);
    const poster = movie.poster_url || 'https://via.placeholder.com/300x450/1e242b/8996a6?text=No+Poster';

    return `
      <div class="poster-hover group relative bg-brand-card rounded-xl overflow-hidden border border-brand-border/70 hover:border-brand-green/60 transition-all duration-300 hover:shadow-xl hover:shadow-brand-green/10 flex flex-col">
        <!-- Poster & Hover Overlay -->
        <div class="relative aspect-[2/3] w-full overflow-hidden bg-brand-dark cursor-pointer" onclick="openMovieDetail(${movie.id})">
          <img src="${poster}" alt="${movie.title}" loading="lazy" class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300" onerror="this.src='https://via.placeholder.com/300x450/1e242b/8996a6?text=Poster'" />
          
          <!-- Rating badge -->
          <div class="absolute top-2.5 right-2.5 px-2 py-0.5 rounded-md bg-black/80 backdrop-blur text-xs font-bold text-amber-400 flex items-center gap-1 border border-white/10">
            ★ <span>${movie.rating || 'N/A'}</span>
          </div>

          <!-- Quick Action Buttons Overlay -->
          <div class="poster-overlay absolute inset-0 bg-gradient-to-t from-black/90 via-black/40 to-transparent opacity-0 transition-opacity duration-200 flex flex-col justify-end p-3 space-y-2">
            <button onclick="event.stopPropagation(); toggleWatchlist(${movie.id})" class="w-full py-1.5 px-2 rounded-lg text-xs font-semibold flex items-center justify-center gap-1.5 transition ${
              isWatchlisted ? 'bg-rose-600 hover:bg-rose-500 text-white' : 'bg-brand-surface/90 hover:bg-brand-green hover:text-black text-white border border-brand-border'
            }">
              <i data-lucide="${isWatchlisted ? 'check' : 'bookmark'}" class="w-3.5 h-3.5"></i>
              <span>${isWatchlisted ? 'In Watchlist' : 'Add Watchlist'}</span>
            </button>
            <button onclick="event.stopPropagation(); openWatchedModal(${movie.id})" class="w-full py-1.5 px-2 rounded-lg text-xs font-semibold bg-brand-surface/90 hover:bg-amber-400 hover:text-black text-white border border-brand-border flex items-center justify-center gap-1.5 transition">
              <i data-lucide="star" class="w-3.5 h-3.5"></i>
              <span>Mark Watched</span>
            </button>
          </div>
        </div>

        <!-- Info Bottom -->
        <div class="p-3 flex-1 flex flex-col justify-between">
          <div>
            <h3 class="font-bold text-sm text-white line-clamp-1 group-hover:text-brand-green transition cursor-pointer" onclick="openMovieDetail(${movie.id})">${movie.title}</h3>
            <p class="text-xs text-brand-muted mt-0.5">${movie.year} • ${movie.director || 'Unknown'}</p>
          </div>
        </div>
      </div>
    `;
  }).join('');

  lucide.createIcons();
}

function isMovieInWatchlist(movieId) {
  return state.watchlist.some(item => {
    const id = item.movie_id || item.movieId || item.id || item;
    return String(id) === String(movieId);
  });
}

// ==========================================
// 6. MOVIE DETAIL MODAL
// ==========================================
async function openMovieDetail(movieId) {
  let movie = state.allMovies.find(m => String(m.id) === String(movieId));

  // If detail isn't full, try calling backend: GET /api/movies/:id
  try {
    const remoteMovie = await apiFetch(`/movies/${movieId}`);
    if (remoteMovie) movie = remoteMovie.data || remoteMovie;
  } catch (err) {
    // If endpoint not ready, fall back to existing movie in state
  }

  if (!movie) return;
  state.activeDetailMovie = movie;

  const isWatchlisted = isMovieInWatchlist(movie.id);
  const poster = movie.poster_url || 'https://via.placeholder.com/300x450';
  let genres = Array.isArray(movie.genres) ? movie.genres : (movie.genres ? movie.genres.split(',') : []);
  let cast = Array.isArray(movie.cast) ? movie.cast : (movie.cast_members ? movie.cast_members.split(',') : []);

  const contentEl = document.getElementById('modal-detail-content');
  contentEl.innerHTML = `
    <div class="flex flex-col sm:flex-row gap-6">
      <!-- Poster -->
      <div class="w-full sm:w-52 shrink-0">
        <img src="${poster}" alt="${movie.title}" class="w-full rounded-xl shadow-xl border border-brand-border" />
      </div>

      <!-- Info -->
      <div class="flex-1 space-y-4">
        <div>
          <div class="flex items-center gap-2 mb-1">
            <span class="text-xs font-bold px-2 py-0.5 rounded bg-brand-green/20 text-brand-green border border-brand-green/30">${movie.year}</span>
            <span class="text-xs text-brand-muted">${movie.duration || ''}</span>
            <span class="text-xs font-bold text-amber-400 flex items-center gap-0.5 ml-auto">★ ${movie.rating || 'N/A'}</span>
          </div>
          <h2 class="text-2xl sm:text-3xl font-black text-white leading-tight">${movie.title}</h2>
          <p class="text-sm text-brand-muted mt-1">Directed by <span class="text-white font-medium">${movie.director || 'Unknown'}</span></p>
        </div>

        <!-- Genres Chips -->
        <div class="flex flex-wrap gap-1.5">
          ${genres.map(g => `<span class="px-2 py-0.5 rounded-full bg-brand-dark border border-brand-border text-[11px] text-brand-muted">${g.trim()}</span>`).join('')}
        </div>

        <!-- Synopsis -->
        <div>
          <h4 class="text-xs font-bold text-brand-muted uppercase tracking-wider mb-1">Synopsis</h4>
          <p class="text-sm text-gray-300 leading-relaxed">${movie.synopsis || 'No synopsis available.'}</p>
        </div>

        <!-- Cast -->
        ${cast.length ? `
          <div>
            <h4 class="text-xs font-bold text-brand-muted uppercase tracking-wider mb-1">Cast</h4>
            <p class="text-xs text-brand-muted">${cast.map(c => c.trim()).join(' • ')}</p>
          </div>
        ` : ''}

        <!-- Actions -->
        <div class="pt-4 border-t border-brand-border flex items-center gap-3">
          <button onclick="toggleWatchlist(${movie.id})" class="flex-1 py-2.5 px-4 rounded-xl font-bold text-xs flex items-center justify-center gap-2 transition ${
            isWatchlisted ? 'bg-rose-600 hover:bg-rose-500 text-white' : 'bg-brand-green text-black hover:brightness-110'
          }">
            <i data-lucide="${isWatchlisted ? 'check' : 'bookmark'}" class="w-4 h-4"></i>
            <span>${isWatchlisted ? 'Remove from Watchlist' : 'Add to Watchlist'}</span>
          </button>

          <button onclick="closeMovieDetailModal(); openWatchedModal(${movie.id})" class="py-2.5 px-4 rounded-xl font-bold text-xs bg-brand-surfaceHover hover:bg-amber-400 hover:text-black border border-brand-border text-white flex items-center justify-center gap-2 transition">
            <i data-lucide="star" class="w-4 h-4"></i>
            <span>Log Review</span>
          </button>
        </div>
      </div>
    </div>
  `;

  const modal = document.getElementById('movie-detail-modal');
  modal.classList.remove('hidden');
  setTimeout(() => modal.classList.remove('opacity-0'), 10);
  lucide.createIcons();
}

function closeMovieDetailModal() {
  const modal = document.getElementById('movie-detail-modal');
  modal.classList.add('opacity-0');
  setTimeout(() => modal.classList.add('hidden'), 200);
}

// ==========================================
// 7. WATCHLIST LOGIC
// ==========================================
async function loadWatchlist() {
  if (!state.token) return;

  try {
    const data = await apiFetch('/watchlist');
    state.watchlist = Array.isArray(data) ? data : (data.data || data.watchlist || []);
    
    // Update badge count
    const badge = document.getElementById('watchlist-badge');
    const countEl = document.getElementById('watchlist-count');
    if (badge) {
      badge.textContent = state.watchlist.length;
      badge.classList.toggle('hidden', state.watchlist.length === 0);
    }
    if (countEl) countEl.textContent = `${state.watchlist.length} films`;

    renderWatchlistGrid();
  } catch (err) {
    console.error('Failed to load watchlist:', err.message);
  }
}

async function toggleWatchlist(movieId) {
  if (!state.token) {
    showToast('Silakan login terlebih dahulu untuk mengelola Watchlist', 'info');
    openAuthModal('login');
    return;
  }

  const isCurrentlyIn = isMovieInWatchlist(movieId);

  try {
    if (isCurrentlyIn) {
      // DELETE /api/watchlist/:movieId
      await apiFetch(`/watchlist/${movieId}`, { method: 'DELETE' });
      state.watchlist = state.watchlist.filter(item => {
        const id = item.movie_id || item.movieId || item.id || item;
        return String(id) !== String(movieId);
      });
      showToast('Film dihapus dari watchlist', 'info');
    } else {
      // POST /api/watchlist
      const res = await apiFetch('/watchlist', {
        method: 'POST',
        body: JSON.stringify({ movieId: movieId, movie_id: movieId })
      });
      state.watchlist.push(res.data || { movieId });
      showToast('Berhasil ditambahkan ke watchlist! 📌', 'success');
    }

    // Refresh UI
    applyMovieFilters();
    loadWatchlist();
    if (state.activeDetailMovie && String(state.activeDetailMovie.id) === String(movieId)) {
      openMovieDetail(movieId);
    }
  } catch (err) {
    showToast(err.message, 'error');
  }
}

function renderWatchlistGrid() {
  const gridEl = document.getElementById('watchlist-grid');
  const emptyEl = document.getElementById('watchlist-empty');
  if (!gridEl) return;

  if (state.watchlist.length === 0) {
    gridEl.innerHTML = '';
    emptyEl.classList.remove('hidden');
    return;
  }

  emptyEl.classList.add('hidden');
  gridEl.innerHTML = state.watchlist.map(item => {
    // Watchlist item might be movie object or { movieId: 1, movie: {...} }
    const movieId = item.movie_id || item.movieId || item.id || item;
    const movie = item.movie || state.allMovies.find(m => String(m.id) === String(movieId)) || {
      id: movieId,
      title: item.title || `Movie #${movieId}`,
      year: item.year || '',
      rating: item.rating || '',
      poster_url: item.poster_url || ''
    };

    const poster = movie.poster_url || 'https://via.placeholder.com/300x450';

    return `
      <div class="bg-brand-card rounded-xl overflow-hidden border border-brand-border flex flex-col group">
        <div class="relative aspect-[2/3] overflow-hidden bg-brand-dark cursor-pointer" onclick="openMovieDetail(${movie.id})">
          <img src="${poster}" alt="${movie.title}" class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300" />
        </div>
        <div class="p-3 flex-1 flex flex-col justify-between">
          <div>
            <h4 class="font-bold text-sm text-white truncate cursor-pointer hover:text-brand-green" onclick="openMovieDetail(${movie.id})">${movie.title}</h4>
            <p class="text-xs text-brand-muted mt-0.5">${movie.year || ''}</p>
          </div>
          <div class="flex items-center gap-2 mt-3 pt-2 border-t border-brand-border">
            <button onclick="toggleWatchlist(${movie.id})" class="text-xs text-rose-400 hover:text-rose-300 py-1 flex items-center gap-1">
              <i data-lucide="trash-2" class="w-3.5 h-3.5"></i> Hapus
            </button>
            <button onclick="openWatchedModal(${movie.id})" class="text-xs text-amber-400 hover:text-amber-300 py-1 ml-auto flex items-center gap-1 font-semibold">
              <i data-lucide="check-circle" class="w-3.5 h-3.5"></i> Watched
            </button>
          </div>
        </div>
      </div>
    `;
  }).join('');

  lucide.createIcons();
}

// ==========================================
// 8. WATCHED & REVIEWS
// ==========================================
async function loadWatchedList() {
  if (!state.token) return;

  try {
    const data = await apiFetch('/watched');
    state.watchedList = Array.isArray(data) ? data : (data.data || data.watched || []);
    renderWatchedList();
  } catch (err) {
    console.error('Failed to load watched list:', err.message);
  }
}

function openWatchedModal(movieId) {
  if (!state.token) {
    showToast('Silakan login terlebih dahulu untuk mencatat film yang sudah ditonton', 'info');
    openAuthModal('login');
    return;
  }

  const movie = state.allMovies.find(m => String(m.id) === String(movieId)) || { id: movieId, title: `Movie #${movieId}` };

  const contentEl = document.getElementById('modal-detail-content');
  contentEl.innerHTML = `
    <div class="space-y-4">
      <div class="border-b border-brand-border pb-3">
        <span class="text-xs font-bold text-brand-green uppercase tracking-wider">Log Review & Thoughts</span>
        <h2 class="text-2xl font-bold text-white mt-1">${movie.title}</h2>
      </div>

      <form id="watched-form" onsubmit="handleWatchedSubmit(event, ${movie.id})" class="space-y-4">
        <div>
          <label class="block text-xs font-semibold text-brand-muted uppercase mb-1.5">Rating Kamu (1 - 5 Bintang)</label>
          <div class="flex items-center gap-2" id="star-selector">
            ${[1, 2, 3, 4, 5].map(star => `
              <button type="button" onclick="setRatingValue(${star})" class="star-btn text-2xl text-brand-muted hover:text-amber-400 transition" data-star="${star}">
                ★
              </button>
            `).join('')}
          </div>
          <input type="hidden" id="watched-rating-input" value="5" />
        </div>

        <div>
          <label class="block text-xs font-semibold text-brand-muted uppercase mb-1.5">Opini & Review Kamu</label>
          <textarea id="watched-review-input" rows="4" required placeholder="Tuliskan pendapatmu tentang alur cerita, akting, atau kesan setelah menonton film ini..." class="w-full bg-brand-dark border border-brand-border rounded-xl p-3 text-sm text-white placeholder-brand-muted focus:outline-none focus:border-brand-green resize-none"></textarea>
        </div>

        <div class="pt-3 border-t border-brand-border flex items-center justify-end gap-3">
          <button type="button" onclick="closeMovieDetailModal()" class="px-4 py-2 text-sm text-brand-muted hover:text-white transition">Batal</button>
          <button type="submit" class="px-5 py-2.5 bg-brand-green text-black font-bold text-sm rounded-xl hover:brightness-110 transition flex items-center gap-2">
            <i data-lucide="check" class="w-4 h-4"></i> Simpan Catatan
          </button>
        </div>
      </form>
    </div>
  `;

  setRatingValue(5);
  const modal = document.getElementById('movie-detail-modal');
  modal.classList.remove('hidden');
  setTimeout(() => modal.classList.remove('opacity-0'), 10);
  lucide.createIcons();
}

function setRatingValue(val) {
  const input = document.getElementById('watched-rating-input');
  if (input) input.value = val;

  document.querySelectorAll('#star-selector .star-btn').forEach(btn => {
    const starNum = parseInt(btn.getAttribute('data-star'));
    if (starNum <= val) {
      btn.className = 'star-btn text-2xl text-amber-400 transition scale-110';
    } else {
      btn.className = 'star-btn text-2xl text-brand-muted transition';
    }
  });
}

async function handleWatchedSubmit(event, movieId) {
  event.preventDefault();
  const rating = parseFloat(document.getElementById('watched-rating-input').value);
  const review = document.getElementById('watched-review-input').value.trim();

  try {
    // POST /api/watched
    await apiFetch('/watched', {
      method: 'POST',
      body: JSON.stringify({
        movieId: movieId,
        movie_id: movieId,
        rating: rating,
        review: review,
        opinion: review
      })
    });

    showToast('Catatan film berhasil disimpan! ⭐', 'success');
    closeMovieDetailModal();
    loadWatchedList();
  } catch (err) {
    showToast(err.message, 'error');
  }
}

function renderWatchedList() {
  const container = document.getElementById('watched-list');
  const emptyEl = document.getElementById('watched-empty');
  if (!container) return;

  if (state.watchedList.length === 0) {
    container.innerHTML = '';
    emptyEl.classList.remove('hidden');
    return;
  }

  emptyEl.classList.add('hidden');
  container.innerHTML = state.watchedList.map(item => {
    const movieId = item.movie_id || item.movieId || (item.movie && item.movie.id);
    const movie = item.movie || state.allMovies.find(m => String(m.id) === String(movieId)) || {
      title: item.title || `Movie #${movieId}`,
      year: item.year || '',
      poster_url: item.poster_url || ''
    };

    const ratingStars = '★'.repeat(Math.round(item.rating || 5)) + '☆'.repeat(Math.max(0, 5 - Math.round(item.rating || 5)));
    const poster = movie.poster_url || 'https://via.placeholder.com/300x450';
    const dateStr = item.created_at || item.createdAt ? new Date(item.created_at || item.createdAt).toLocaleDateString('id-ID', { dateStyle: 'medium' }) : 'Baru saja';

    return `
      <div class="bg-brand-surface p-4 sm:p-5 rounded-2xl border border-brand-border flex gap-4 sm:gap-6 items-start">
        <img src="${poster}" alt="${movie.title}" class="w-16 sm:w-20 aspect-[2/3] object-cover rounded-lg shrink-0 border border-brand-border shadow-md" />
        <div class="flex-1 space-y-2">
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-1">
            <h4 class="font-bold text-base text-white hover:text-brand-green cursor-pointer" onclick="openMovieDetail(${movieId})">
              ${movie.title} <span class="text-xs text-brand-muted font-normal">(${movie.year || ''})</span>
            </h4>
            <div class="flex items-center gap-2">
              <span class="text-amber-400 font-bold text-sm tracking-wider">${ratingStars}</span>
              <span class="text-[11px] text-brand-muted">${dateStr}</span>
            </div>
          </div>
          <p class="text-sm text-gray-300 leading-relaxed bg-brand-dark/40 p-3 rounded-xl border border-brand-border/40 font-normal">
            "${item.review || item.opinion || 'Tidak ada catatan ulasan.'}"
          </p>
        </div>
      </div>
    `;
  }).join('');
}

// ==========================================
// 9. AI RECOMMENDATION (GEMINI)
// ==========================================
function setAiPrompt(text) {
  document.getElementById('ai-prompt-input').value = text;
}

async function submitAiRecommendation() {
  const promptInput = document.getElementById('ai-prompt-input');
  const prompt = promptInput.value.trim();
  if (!prompt) {
    showToast('Tuliskan deskripsi atau suasana film yang ingin kamu cari', 'warning');
    return;
  }

  const btn = document.getElementById('btn-submit-ai');
  const resultContainer = document.getElementById('ai-result-container');
  
  btn.disabled = true;
  btn.innerHTML = `<div class="animate-spin w-4 h-4 border-2 border-black border-t-transparent rounded-full"></div> Analyzing...`;

  try {
    // POST /api/movies/recommend
    const response = await apiFetch('/movies/recommend', {
      method: 'POST',
      body: JSON.stringify({ prompt: prompt, query: prompt, description: prompt })
    });

    const resultData = response.data || response;
    renderAiRecommendation(resultData);
    resultContainer.classList.remove('hidden');
    showToast('Gemini berhasil menemukan film yang cocok! ✨', 'success');
  } catch (err) {
    showToast(`AI Recommendation Error: ${err.message}`, 'error');
  } finally {
    btn.disabled = false;
    btn.innerHTML = `<i data-lucide="sparkles" class="w-4 h-4"></i> Ask AI`;
    lucide.createIcons();
  }
}

function renderAiRecommendation(data) {
  const container = document.getElementById('ai-result-container');
  
  // Backend might return { movie_id: 1, reasoning: "..." } or full { movie: {...}, reasoning: "..." }
  const movieId = data.movie_id || data.movieId || (data.movie && data.movie.id);
  const matchedMovie = data.movie || state.allMovies.find(m => String(m.id) === String(movieId)) || {
    id: movieId || 1,
    title: data.title || 'Recommended Film',
    year: data.year || '',
    genres: data.genres || ['Recommended'],
    rating: data.rating || 8.5,
    synopsis: data.synopsis || '',
    poster_url: data.poster_url || 'https://via.placeholder.com/300x450'
  };

  const reasoning = data.reasoning || data.explanation || data.message || 'Film ini dipilih berdasarkan kecocokan tema, plot, dan atmosfer yang Anda minta.';
  const poster = matchedMovie.poster_url || 'https://via.placeholder.com/300x450';

  container.innerHTML = `
    <div class="bg-gradient-to-br from-brand-surface via-brand-card to-brand-surface p-6 sm:p-8 rounded-2xl border-2 border-brand-green/40 shadow-2xl relative overflow-hidden">
      <div class="absolute top-0 right-0 p-8 opacity-10 pointer-events-none">
        <i data-lucide="sparkles" class="w-32 h-32 text-brand-green"></i>
      </div>

      <div class="relative z-10">
        <div class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-brand-green/20 text-brand-green text-xs font-bold mb-4 border border-brand-green/40">
          ✨ Rekomendasi Terpilih dari Database Anda
        </div>

        <div class="flex flex-col sm:flex-row gap-6 items-start">
          <img src="${poster}" alt="${matchedMovie.title}" class="w-36 sm:w-44 aspect-[2/3] object-cover rounded-xl border border-brand-border shadow-xl shrink-0" />
          
          <div class="flex-1 space-y-3">
            <div>
              <div class="flex items-center gap-2">
                <span class="text-xs font-bold text-amber-400">★ ${matchedMovie.rating || 'N/A'}</span>
                <span class="text-xs text-brand-muted">• ${matchedMovie.year || ''}</span>
              </div>
              <h3 class="text-2xl font-black text-white hover:text-brand-green cursor-pointer" onclick="openMovieDetail(${matchedMovie.id})">${matchedMovie.title}</h3>
              <p class="text-xs text-brand-muted">Directed by ${matchedMovie.director || 'Unknown'}</p>
            </div>

            <!-- Gemini Reasoning Card -->
            <div class="bg-brand-dark/70 border border-brand-border/60 rounded-xl p-4 space-y-1.5">
              <span class="text-xs font-bold text-brand-green flex items-center gap-1.5">
                <i data-lucide="bot" class="w-3.5 h-3.5"></i> Alasan Rekomendasi (Gemini Analysis):
              </span>
              <p class="text-xs text-gray-200 leading-relaxed italic">
                "${reasoning}"
              </p>
            </div>

            <div class="pt-2 flex items-center gap-3">
              <button onclick="openMovieDetail(${matchedMovie.id})" class="px-4 py-2 bg-brand-surfaceHover border border-brand-border hover:border-brand-green text-xs font-semibold text-white rounded-xl transition">
                Lihat Detail Lengkap
              </button>
              <button onclick="toggleWatchlist(${matchedMovie.id})" class="px-4 py-2 bg-brand-green text-black text-xs font-bold rounded-xl hover:brightness-110 transition flex items-center gap-1.5">
                <i data-lucide="bookmark" class="w-3.5 h-3.5"></i> Masukkan ke Watchlist
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  `;

  lucide.createIcons();
}

// ==========================================
// 10. AUTHENTICATION (LOGIN & REGISTER)
// ==========================================
function renderAuthUI() {
  const container = document.getElementById('auth-container');
  if (!container) return;

  if (state.token && state.user) {
    container.innerHTML = `
      <div class="flex items-center space-x-2">
        <div class="w-8 h-8 rounded-full bg-brand-border flex items-center justify-center font-bold text-xs text-white uppercase border border-brand-border">
          ${(state.user.username || state.user.email || 'U')[0]}
        </div>
        <span class="text-xs font-semibold text-white hidden sm:inline">${state.user.username || state.user.email}</span>
        <button onclick="logout()" title="Logout" class="p-1.5 rounded-lg text-brand-muted hover:text-rose-400 hover:bg-rose-950/20 transition">
          <i data-lucide="log-out" class="w-4 h-4"></i>
        </button>
      </div>
    `;
  } else {
    container.innerHTML = `
      <div class="flex items-center space-x-2">
        <button onclick="openAuthModal('login')" class="px-3 py-1.5 rounded-lg text-xs font-semibold text-white hover:bg-brand-surfaceHover transition">
          Login
        </button>
        <button onclick="openAuthModal('register')" class="px-3 py-1.5 rounded-lg text-xs font-bold bg-brand-green text-black hover:brightness-110 transition">
          Register
        </button>
      </div>
    `;
  }
  lucide.createIcons();
}

function openAuthModal(mode = 'login') {
  setAuthMode(mode);
  const modal = document.getElementById('auth-modal');
  modal.classList.remove('hidden');
  setTimeout(() => modal.classList.remove('opacity-0'), 10);
}

function closeAuthModal() {
  const modal = document.getElementById('auth-modal');
  modal.classList.add('opacity-0');
  setTimeout(() => modal.classList.add('hidden'), 200);
}

function setAuthMode(mode) {
  state.authMode = mode;
  const usernameField = document.getElementById('field-username-container');
  const submitBtn = document.getElementById('auth-submit-btn');
  const tabLogin = document.getElementById('auth-tab-login');
  const tabRegister = document.getElementById('auth-tab-register');
  const errorMsg = document.getElementById('auth-error-msg');
  errorMsg.classList.add('hidden');

  if (mode === 'register') {
    usernameField.classList.remove('hidden');
    submitBtn.textContent = 'Create Account';
    tabRegister.className = 'font-bold text-lg text-white border-b-2 border-brand-green pb-1';
    tabLogin.className = 'font-medium text-lg text-brand-muted hover:text-white pb-1';
  } else {
    usernameField.classList.add('hidden');
    submitBtn.textContent = 'Sign In';
    tabLogin.className = 'font-bold text-lg text-white border-b-2 border-brand-green pb-1';
    tabRegister.className = 'font-medium text-lg text-brand-muted hover:text-white pb-1';
  }
}

async function handleAuthSubmit(event) {
  event.preventDefault();
  const email = document.getElementById('auth-email').value.trim();
  const password = document.getElementById('auth-password').value;
  const username = document.getElementById('auth-username').value.trim();
  const errorEl = document.getElementById('auth-error-msg');
  const submitBtn = document.getElementById('auth-submit-btn');

  errorEl.classList.add('hidden');
  submitBtn.disabled = true;
  submitBtn.textContent = 'Processing...';

  try {
    const endpoint = state.authMode === 'register' ? '/auth/register' : '/auth/login';
    const payload = state.authMode === 'register' 
      ? { email, password, username: username || email.split('@')[0] } 
      : { email, password };

    const data = await apiFetch(endpoint, {
      method: 'POST',
      body: JSON.stringify(payload)
    });

    // Handle token format: could be { token: "..." } or { data: { token: "...", user: {...} } }
    const token = data.token || (data.data && data.data.token) || (data.user && data.user.token);
    const user = data.user || (data.data && data.data.user) || { email, username: username || email };

    if (token) {
      state.token = token;
      state.user = user;
      localStorage.setItem('filmlog_token', token);
      localStorage.setItem('filmlog_user', JSON.stringify(user));
      showToast(`Berhasil masuk sebagai ${user.username || user.email}! 👋`, 'success');
      closeAuthModal();
      renderAuthUI();
      loadWatchlist();
      loadWatchedList();
      applyMovieFilters();
    } else {
      showToast('Registrasi berhasil! Silakan login.', 'success');
      setAuthMode('login');
    }
  } catch (err) {
    errorEl.textContent = err.message;
    errorEl.classList.remove('hidden');
  } finally {
    submitBtn.disabled = false;
    submitBtn.textContent = state.authMode === 'register' ? 'Create Account' : 'Sign In';
  }
}

function logout() {
  state.token = null;
  state.user = null;
  state.watchlist = [];
  state.watchedList = [];
  localStorage.removeItem('filmlog_token');
  localStorage.removeItem('filmlog_user');
  renderAuthUI();
  applyMovieFilters();
  switchTab('discover');
  showToast('Anda telah keluar (Logged out)', 'info');
}

// ==========================================
// 11. API CONFIGURATION MODAL
// ==========================================
function openApiConfigModal() {
  const modal = document.getElementById('api-config-modal');
  modal.classList.remove('hidden');
  setTimeout(() => modal.classList.remove('opacity-0'), 10);
}

function closeApiConfigModal() {
  const modal = document.getElementById('api-config-modal');
  modal.classList.add('opacity-0');
  setTimeout(() => modal.classList.add('hidden'), 200);
}

function setApiUrlPreset(url) {
  document.getElementById('api-base-url-input').value = url;
}

function saveApiBaseUrl() {
  const input = document.getElementById('api-base-url-input');
  let val = input.value.trim().replace(/\/$/, '');
  if (!val.startsWith('http://') && !val.startsWith('https://')) {
    val = 'http://' + val;
  }
  state.apiBaseUrl = val;
  localStorage.setItem('filmlog_api_url', val);
  updateApiUrlDisplay();
  closeApiConfigModal();
  showToast(`API URL diubah ke: ${val}`, 'success');
  loadDiscoverMovies();
  if (state.token) {
    loadWatchlist();
    loadWatchedList();
  }
}

// ==========================================
// 12. TOAST NOTIFICATIONS
// ==========================================
let toastTimeout = null;
function showToast(message, type = 'info') {
  const toast = document.getElementById('toast');
  const icon = document.getElementById('toast-icon');
  const msg = document.getElementById('toast-message');

  const icons = {
    success: '🟢',
    error: '🔴',
    warning: '🟡',
    info: '🔵'
  };

  icon.textContent = icons[type] || '🔵';
  msg.textContent = message;

  toast.classList.remove('translate-y-20', 'opacity-0');
  toast.classList.add('translate-y-0', 'opacity-100');

  if (toastTimeout) clearTimeout(toastTimeout);
  toastTimeout = setTimeout(() => {
    toast.classList.remove('translate-y-0', 'opacity-100');
    toast.classList.add('translate-y-20', 'opacity-0');
  }, 3500);
}
