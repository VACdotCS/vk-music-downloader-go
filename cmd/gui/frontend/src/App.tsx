import { useState, useEffect } from 'react';
import { HasValidToken, SaveToken, SelectDirectory, GetSavePath, DownloadAllAudio, DownloadTrack, DownloadPlaylist } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';
import './App.css';

export default function App() {
  const [hasToken, setHasToken] = useState(false);
  const [loading, setLoading] = useState(true);
  const [tokenInput, setTokenInput] = useState('');
  const [savePath, setSavePath] = useState('');
  const [isDownloading, setIsDownloading] = useState(false);
  const [progressLog, setProgressLog] = useState<{ [key: number]: any }>({});
  
  // URL inputs
  const [trackUrl, setTrackUrl] = useState('');
  const [playlistUrl, setPlaylistUrl] = useState('');

  useEffect(() => {
    checkToken();
    EventsOn('download-progress', (data) => {
      setProgressLog((prev) => ({ ...prev, [data.index]: data }));
    });
  }, []);

  const checkToken = async () => {
    const valid = await HasValidToken();
    setHasToken(valid);
    if (valid) {
      const path = await GetSavePath();
      setSavePath(path || 'Не выбрана');
    }
    setLoading(false);
  };

  const handleSaveToken = async () => {
    try {
      await SaveToken(tokenInput);
      checkToken();
    } catch (e) {
      alert('Ошибка: Неверный формат токена. Ожидался JSON.');
    }
  };

  const handleSelectDir = async () => {
    const path = await SelectDirectory();
    if (path) setSavePath(path);
  };

  const runDownload = async (fn: () => Promise<void>) => {
    if (!savePath || savePath === 'Не выбрана') {
      alert('Сначала выберите папку для сохранения!');
      return;
    }
    setIsDownloading(true);
    setProgressLog({});
    try {
      await fn();
      alert('Скачивание успешно завершено!');
    } catch (e) {
      alert('Ошибка: ' + e);
    }
    setIsDownloading(false);
  };

  if (loading) return <div className="app-container"><div className="loader"></div></div>;

  if (!hasToken) {
    return (
      <div className="app-container flex-center">
        <div className="card">
          <h1>Вход в VK Music</h1>
          <p>Вставьте JSON-объект с вашим access_token из ВК:</p>
          <textarea
            value={tokenInput}
            onChange={(e) => setTokenInput(e.target.value)}
            placeholder='{"data": {"access_token": "...", ...}}'
          />
          <button className="btn-primary" onClick={handleSaveToken}>Сохранить токен</button>
        </div>
      </div>
    );
  }

  if (isDownloading) {
    const items = Object.values(progressLog).sort((a, b) => a.index - b.index);
    return (
      <div className="app-container">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <h2>Прогресс скачивания</h2>
        </div>
        <div className="progress-list">
          {items.map((item) => (
            <div key={item.index} className={`progress-item ${item.status === 'error' ? 'item-error' : ''}`}>
              <span className="track-title">{item.index}. {item.title}</span>
              {item.status === 'error' ? (
                <span className="error-text">❌ Ошибка скачивания</span>
              ) : (
                <div className="progress-bar-bg">
                  <div 
                    className={`progress-bar-fill ${item.status === 'done' ? 'done' : ''}`}
                    style={{ width: `${item.percentage * 100}%` }}
                  ></div>
                </div>
              )}
            </div>
          ))}
          {items.length === 0 && <p className="loading-text">Загрузка данных...</p>}
        </div>
      </div>
    );
  }

  return (
    <div className="app-container">
      <div className="header">
        <h1>Меню загрузки</h1>
      </div>
      
      <div className="card menu-card">
        <div className="menu-item">
          <div>
            <h3>Папка для сохранения</h3>
            <p className="path-text">{savePath}</p>
          </div>
          <button className="btn-secondary" onClick={handleSelectDir}>Изменить</button>
        </div>

        <div className="scenario-section">
          <h3>🎵 Все треки</h3>
          <button className="btn-action" onClick={() => runDownload(DownloadAllAudio)}>
            Скачать всю мою музыку
          </button>
        </div>

        <div className="scenario-section">
          <h3>▶️ Скачать плейлист</h3>
          <div className="input-group">
            <input 
              type="text" 
              placeholder="Вставьте ссылку на плейлист..." 
              value={playlistUrl}
              onChange={(e) => setPlaylistUrl(e.target.value)}
            />
            <button className="btn-primary" onClick={() => runDownload(() => DownloadPlaylist(playlistUrl))}>
              Скачать
            </button>
          </div>
        </div>

        <div className="scenario-section">
          <h3>🎶 Один трек</h3>
          <div className="input-group">
            <input 
              type="text" 
              placeholder="Вставьте ссылку на трек..." 
              value={trackUrl}
              onChange={(e) => setTrackUrl(e.target.value)}
            />
            <button className="btn-primary" onClick={() => runDownload(() => DownloadTrack(trackUrl))}>
              Скачать
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
