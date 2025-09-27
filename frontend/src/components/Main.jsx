import React, { useState, useRef, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import Cookies from 'js-cookie';
import './Main.css';

const Main = () => {
  const [input, setInput] = useState('');
  const [output, setOutput] = useState(['']);
  const [isExecuting, setIsExecuting] = useState(false);
  const fileInputRef = useRef(null);
  const terminalRef = useRef(null);
  const inputRef = useRef(null);
  const [loading, setLoading] = useState(false);
  const usuario = Cookies.get('usuario');
  const navigate = useNavigate();

  useEffect(() => {
    if (terminalRef.current) {
      terminalRef.current.scrollTop = terminalRef.current.scrollHeight;
    }
  }, [output]);

  const handleFileChange = (event) => {
    const file = event.target.files[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = (e) => {
      setInput(e.target.result);
    };
    reader.readAsText(file);
  };

  const handleChooseFileClick = () => {
    fileInputRef.current.click();
  };

  const handleLogout = () => {
    setLoading(true);
    setTimeout(() => {
      Cookies.remove('usuario');
      setLoading(false);
      navigate('/main');
    }, 2000);
  };

  const handleLogin = () => {
    setLoading(true);
    setTimeout(() => {
      setLoading(false);
      navigate('/login');
    }, 2000);
  };

  const executeCommand = async () => {
    if (!input.trim()) return;

    const command = input.trim();
    setIsExecuting(true);
    setOutput(prev => [...prev, `$ ${command}`]);

    const lowerCmd = command.toLowerCase();
    const knownCommands = ['help', 'clear', 'status', 'users', 'disk'];

    if (lowerCmd === 'clear') {
      setOutput(['']);
      setIsExecuting(false);
      setInput('');
      return;
    }

    try {
      const response = await fetch('http://localhost:8000/execute', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ comando: command }),
      });

      const data = await response.json();
      setOutput(prev => [...prev, data.salida || 'Sin respuesta del servidor']);
    } catch (error) {
      setOutput(prev => [...prev, `Error al conectar con el backend: ${error.message}`]);
    }

    setIsExecuting(false);
    setInput('');
  };

  const handleKeyPress = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      executeCommand();
    }
  };

  return (
    <div className="main-bg">
      {loading && (
        <div className="main-loading-overlay">
          <div className="main-spinner"></div>
          <span className="main-loading-text">Cargando...</span>
        </div>
      )}
      <header className="main-header">
        <div className="main-header-left">
          <div className="main-logo"></div>
          <div>
            <h1 className="main-title">Sistema de Particiones</h1>
            <p className="main-subtitle">Panel Principal</p>
          </div>
        </div>
        <div className="main-header-right">
          {!usuario ? (
            <button onClick={handleLogin} className="main-login-btn">Iniciar Sesión</button>
          ) : (
            <>
              <span className="main-user"> Usuario Actual: {usuario}</span>
              <button onClick={handleLogout} className="main-logout-btn">Cerrar Sesión</button>
              <button onClick={() => navigate('/explorer')} className="main-explorer-btn">Explorador</button>
            </>
          )}
        </div>
      </header>

      <main className="main-content">
        <div className="terminal-section">
          <div className="terminal-label">Terminal</div>

          <div ref={terminalRef} className="terminal-box">
            {output.map((line, idx) => (
              <div key={idx} className="terminal-line">
                {line.startsWith('$') && <span className="terminal-prompt">{usuario}@sistema:</span>}
                <span className={line.startsWith('$') ? 'terminal-command' : 'terminal-output'}>
                  {line}
                </span>
              </div>
            ))}
            {isExecuting && (
              <div className="terminal-processing">
                <span className="terminal-prompt">{usuario}@sistema:</span>
                <span className="cursor-blink">Procesando comando...</span>
              </div>
            )}
            <div className="terminal-cursor">
              <span className="terminal-prompt">{usuario}@sistema:</span>
              <span className="cursor-blink">█</span>
            </div>
          </div>

          <div className="terminal-input-area">
            <textarea
              ref={inputRef}
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyPress}
              placeholder="Escribe tu comando aquí o carga un archivo..."
              className="terminal-input"
              disabled={isExecuting}
            />
            <button
              onClick={executeCommand}
              disabled={isExecuting || !input.trim()}
              className="main-execute-btn"
            >
              {isExecuting ? 'Ejecutando...' : 'Ejecutar'}
            </button>
          </div>
        </div>

        <div className="quick-actions">
          <input
            type="file"
            ref={fileInputRef}
            onChange={handleFileChange}
            style={{ display: 'none' }}
            accept=".smia"
          />
          <button onClick={() => setInput('clear')} className="quick-action-btn">Borrar</button>
          <button onClick={handleChooseFileClick} className="quick-action-btn">Elegir archivo</button>
        </div>
      </main>
    </div>
  );
};

export default Main;
