import { useState, useEffect, useRef } from 'react';
import { HasValidToken, SaveToken, ParseAndSaveUrlToken, OpenAuthPage, SelectDirectory, GetSavePath, DownloadAllAudio, DownloadTrack, DownloadPlaylist, CancelDownload, ClearToken } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';
import './App.css';

// Иконки (встроенные SVG для красоты)
const IconFolder = () => <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path></svg>;
const IconDownload = () => <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="7 10 12 15 17 10"></polyline><line x1="12" y1="15" x2="12" y2="3"></line></svg>;
const IconMusic = () => <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M9 18V5l12-2v13"></path><circle cx="6" cy="18" r="3"></circle><circle cx="18" cy="16" r="3"></circle></svg>;
const IconList = () => <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="8" y1="6" x2="21" y2="6"></line><line x1="8" y1="12" x2="21" y2="12"></line><line x1="8" y1="18" x2="21" y2="18"></line><line x1="3" y1="6" x2="3.01" y2="6"></line><line x1="3" y1="12" x2="3.01" y2="12"></line><line x1="3" y1="18" x2="3.01" y2="18"></line></svg>;
const IconCheck = () => <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#10b981" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>;
const IconX = () => <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#ef4444" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>;

export default function App() {
  const [hasToken, setHasToken] = useState(false);
  const [loading, setLoading] = useState(true);
  const [tokenInput, setTokenInput] = useState('');
  const [authMode, setAuthMode] = useState<'url' | 'json'>('url');
  
  const [savePath, setSavePath] = useState('');
  const [isDownloading, setIsDownloading] = useState(false);
  const [progressLog, setProgressLog] = useState<{ [key: number]: any }>({});
  
  const [trackUrl, setTrackUrl] = useState('');
  const [playlistUrl, setPlaylistUrl] = useState('');
  
  const [autoScroll, setAutoScroll] = useState(true);
  const listRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    checkToken();
    EventsOn('download-progress', (data) => {
      setProgressLog((prev) => ({ ...prev, [data.index]: data }));
      
      // Авто-логаут при протухании токена (400 ошибки)
      if (data.status === 'error-token') {
        alert("Токен или ссылки устарели (Ошибка 400). Пожалуйста, получите новый токен!");
        CancelDownload().then(() => {
          ClearToken().then(() => {
            window.location.reload();
          });
        });
      }
    });
  }, []);

  useEffect(() => {
    if (autoScroll && listRef.current) {
      listRef.current.scrollTop = listRef.current.scrollHeight;
    }
  }, [progressLog, autoScroll]);

  const checkToken = async () => {
    const valid = await HasValidToken();
    setHasToken(valid);
    if (valid) {
      const path = await GetSavePath();
      setSavePath(path || 'Папка не выбрана');
    }
    setLoading(false);
  };

  const handleSaveToken = async () => {
    try {
      await SaveToken(tokenInput);
      checkToken();
    } catch (e) {
      alert('Ошибка: ' + e);
    }
  };

  const handleSelectDir = async () => {
    const path = await SelectDirectory();
    if (path) setSavePath(path);
  };

  const runDownload = async (fn: () => Promise<void>) => {
    if (!savePath || savePath === 'Папка не выбрана') {
      alert('Пожалуйста, выберите папку для сохранения музыки!');
      return;
    }
    setIsDownloading(true);
    setProgressLog({});
    try {
      await fn();
      setTimeout(() => alert('Скачивание завершено!'), 500);
    } catch (e) {
      const errStr = String(e);
      if (errStr.includes('access_token has expired') || errStr.includes('authorization failed')) {
        alert("Токен устарел! Пожалуйста, получите новый токен (через JSON перехват).");
        ClearToken().then(() => window.location.reload());
      } else {
        alert('Ошибка: ' + errStr);
      }
      setIsDownloading(false);
    }
  };

  const cancelDownload = async () => {
    await CancelDownload();
    setIsDownloading(false);
  }

  const handleLogout = async () => {
    await ClearToken();
    checkToken();
  };

  if (loading) return <div className="app-container"><div className="loader"></div></div>;

  if (!hasToken) {
    return (
      <div className="app-container flex-center fade-in">
        <div className="card auth-card">
          <div className="auth-icon"><IconMusic /></div>
          <h1>Вход в VK Music</h1>
          <p className="subtitle">
            Вставьте JSON-объект с вашим access_token из ВК (Kate Mobile):
          </p>

          <textarea
            className="modern-input"
            value={tokenInput}
            onChange={(e) => setTokenInput(e.target.value)}
            placeholder='{"data": {"access_token": "...", ...}}'
            style={{ height: '120px' }}
          />
          <button className="btn-primary auth-btn" onClick={handleSaveToken}>
            Продолжить
          </button>
          
          <div className="auth-footer" style={{ marginTop: '1rem', fontSize: '0.85rem' }}>
            <p className="help-text">
              Способ по OAuth ссылке больше не работает (ВК блокирует скачивание). 
              Используйте сниффер для получения JSON токена.
            </p>
            <p className="help-text">
              <a href="#" onClick={(e) => { e.preventDefault(); alert("Инструкция на GitHub: https://github.com/VACdotCS/vk-music-downloader")}}>Гайд на GitHub</a>
            </p>
          </div>
        </div>
      </div>
    );
  }

  if (isDownloading) {
    const items = Object.values(progressLog).sort((a, b) => a.index - b.index);
    const doneCount = items.filter(i => i.status === 'done' || i.status === 'error').length;
    const totalCount = items.length;
    
    return (
      <div className="app-container fade-in layout-col">
        <div className="header-glass">
          <div>
            <h2>Прогресс загрузки</h2>
            <p className="subtitle">{doneCount} из {totalCount} обработано</p>
          </div>
          <div className="header-actions">
            <label className="toggle-wrapper">
              <input 
                type="checkbox" 
                checked={autoScroll} 
                onChange={(e) => setAutoScroll(e.target.checked)} 
              />
              <span className="toggle-slider"></span>
              <span className="toggle-label">Автоскролл</span>
            </label>
            <button className="btn-secondary btn-sm" onClick={cancelDownload}>Назад</button>
          </div>
        </div>

        <div className="progress-list" ref={listRef}>
          {items.map((item) => (
            <div key={item.index} className={`progress-row ${item.status}`}>
              <div className="progress-row-info">
                <span className="track-number">{item.index}</span>
                <span className="track-title" title={item.title}>{item.title}</span>
              </div>
              
              <div className="progress-row-status">
                {item.status === 'error' && <><IconX /> <span className="text-red">Ошибка</span></>}
                {item.status === 'done' && <><IconCheck /> <span className="text-green">Скачан</span></>}
                {item.status === 'downloading' && (
                  <div className="mini-progress-bar">
                    <div className="mini-progress-fill" style={{ width: `${item.percentage * 100}%` }}></div>
                  </div>
                )}
              </div>
            </div>
          ))}
          {items.length === 0 && (
            <div className="empty-state">
              <div className="spinner"></div>
              <p>Связь с серверами VK...</p>
            </div>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="app-container fade-in">
      <div className="topbar">
        <div className="brand">
          <div className="brand-icon"><IconMusic /></div>
          <h2>VK Music</h2>
        </div>
        <button className="btn-secondary btn-sm" onClick={handleLogout} title="Сменить токен">
          Выйти
        </button>
      </div>
      
      <div className="dashboard-grid">
        {/* Левая колонка */}
        <div className="card save-card">
          <div className="card-header">
            <div className="icon-wrapper"><IconFolder /></div>
            <h3>Куда сохраняем?</h3>
          </div>
          <div className="path-box" title={savePath}>
            {savePath}
          </div>
          <button className="btn-secondary w-full mt-1" onClick={handleSelectDir}>
            Выбрать другую папку
          </button>
        </div>

        {/* Правая колонка - Сценарии */}
        <div className="scenarios-container">
          
          <div className="scenario-card highlight">
            <div className="scenario-info">
              <h3>Моя музыка</h3>
              <p>Скачать все треки, добавленные в ваш профиль ВК</p>
            </div>
            <button className="btn-primary btn-icon" onClick={() => runDownload(DownloadAllAudio)}>
              <IconDownload /> Скачать всё
            </button>
          </div>

          <div className="scenario-card">
            <div className="scenario-info">
              <h3><IconList /> Плейлист</h3>
              <input 
                className="modern-input sm-input"
                type="text" 
                placeholder="https://vk.com/music/playlist/..." 
                value={playlistUrl}
                onChange={(e) => setPlaylistUrl(e.target.value)}
              />
            </div>
            <button className="btn-secondary btn-icon" onClick={() => runDownload(() => DownloadPlaylist(playlistUrl))}>
              <IconDownload /> Загрузить
            </button>
          </div>

          <div className="scenario-card">
            <div className="scenario-info">
              <h3><IconMusic /> Один трек</h3>
              <input 
                className="modern-input sm-input"
                type="text" 
                placeholder="https://vk.com/audio..." 
                value={trackUrl}
                onChange={(e) => setTrackUrl(e.target.value)}
              />
            </div>
            <button className="btn-secondary btn-icon" onClick={() => runDownload(() => DownloadTrack(trackUrl))}>
              <IconDownload /> Загрузить
            </button>
          </div>

        </div>
      </div>
    </div>
  );
}
