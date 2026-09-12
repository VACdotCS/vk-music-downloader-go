import { useState, useEffect } from 'react';
import { HasValidToken, SaveToken, SelectDirectory, GetSavePath, DownloadAllAudio } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';
import './App.css';

export default function App() {
  const [hasToken, setHasToken] = useState(false);
  const [loading, setLoading] = useState(true);
  const [tokenInput, setTokenInput] = useState('');
  const [savePath, setSavePath] = useState('');
  const [isDownloading, setIsDownloading] = useState(false);
  const [progressLog, setProgressLog] = useState<{ [key: number]: any }>({});

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

  const handleDownloadAll = async () => {
    if (!savePath || savePath === 'Не выбрана') {
      alert('Сначала выберите папку для сохранения!');
      return;
    }
    setIsDownloading(true);
    try {
      await DownloadAllAudio();
      alert('Скачивание успешно завершено!');
    } catch (e) {
      alert('Ошибка при скачивании: ' + e);
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
          <p className="help-text">Если вы не знаете как получить токен, ознакомьтесь с гайдом на GitHub.</p>
        </div>
      </div>
    );
  }

  if (isDownloading) {
    const items = Object.values(progressLog).sort((a, b) => a.index - b.index);
    return (
      <div className="app-container">
        <h2>Скачивание треков...</h2>
        <div className="progress-list">
          {items.map((item) => (
            <div key={item.index} className="progress-item">
              <span className="track-title">{item.title}</span>
              <div className="progress-bar-bg">
                <div 
                  className={`progress-bar-fill ${item.percentage >= 1.0 ? 'done' : ''}`}
                  style={{ width: `${item.percentage * 100}%` }}
                ></div>
              </div>
            </div>
          ))}
          {items.length === 0 && <p>Получение списка треков...</p>}
        </div>
      </div>
    );
  }

  return (
    <div className="app-container">
      <div className="header">
        <h1>VK Music Downloader</h1>
      </div>
      
      <div className="card menu-card">
        <div className="menu-item">
          <div>
            <h3>Папка для сохранения</h3>
            <p className="path-text">{savePath}</p>
          </div>
          <button className="btn-secondary" onClick={handleSelectDir}>Изменить</button>
        </div>

        <div className="actions">
          <button className="btn-action" onClick={handleDownloadAll}>
            🎵 Скачать все треки
          </button>
        </div>
      </div>
    </div>
  );
}
