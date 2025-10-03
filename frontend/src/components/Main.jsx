import React, { useState, useRef, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import Cookies from 'js-cookie';
import './Main.css';

const Main = () => {
  const [state, setState] = useState({
    input: '',
    output: [''],
    isExecuting: false,
    loading: false
  });

  const refs = {
    fileInput: useRef(null),
    terminal: useRef(null),
    input: useRef(null)
  };
  const navigate = useNavigate();
  const usuario = Cookies.get('usuario') || 'guest';
  useEffect(() => {
    if (refs.terminal.current) {
      refs.terminal.current.scrollTop = refs.terminal.current.scrollHeight;
    }
  }, [state.output]);

  useEffect(() => {
    if (!state.isExecuting && refs.input.current) {
      refs.input.current.focus();
    }
  }, [state.isExecuting]);

  const handleFileChange = useCallback((event) => {
    const file = event.target.files[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = (e) => {
      setState(prev => ({
        ...prev,
        input: e.target.result
      }));
    };
    reader.readAsText(file);
  }, []);

  const handleChooseFileClick = useCallback(() => {
    refs.fileInput.current?.click();
  }, []);
  const handleLogout = useCallback(() => {
    setState(prev => ({ ...prev, loading: true }));
    
    setTimeout(() => {
      Cookies.remove('usuario');
      setState(prev => ({ ...prev, loading: false }));
      navigate('/main');
    }, 1500);
  }, [navigate]);

  const handleLogin = useCallback(() => {
    setState(prev => ({ ...prev, loading: true }));
    
    setTimeout(() => {
      setState(prev => ({ ...prev, loading: false }));
      navigate('/login');
    }, 1500);
  }, [navigate]);

  const handleExplorer = useCallback(() => {
    navigate('/explorer');
  }, [navigate]);

  const executeCommand = useCallback(async () => {
    if (!state.input.trim()) return;

    const command = state.input.trim();
    
    setState(prev => ({
      ...prev,
      isExecuting: true,
      output: [...prev.output, `$ ${command}`]
    }));

    if (command.toLowerCase() === 'clear') {
      setTimeout(() => {
        setState(prev => ({
          ...prev,
          output: [''],
          isExecuting: false,
          input: ''
        }));
      }, 300);
      return;
    }
    try {
      const response = await fetch('http://localhost:8000/execute', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ comando: command }),
      });

      const data = await response.json();
      
      setTimeout(() => {
        setState(prev => ({
          ...prev,
          output: [...prev.output, data.salida || 'Sin respuesta del servidor'],
          isExecuting: false,
          input: ''
        }));
      }, Math.random() * 800 + 200);

    } catch (error) {
      setTimeout(() => {
        setState(prev => ({
          ...prev,
          output: [...prev.output, `Error: ${error.message}`],
          isExecuting: false,
          input: ''
        }));
      }, 500);
    }
  }, [state.input]);

  const handleKeyPress = useCallback((e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      executeCommand();
    }
    
    if (e.key === 'ArrowUp' && e.ctrlKey) {
      e.preventDefault();
      setState(prev => ({
        ...prev,
        input: 'status'
      }));
    }
  }, [executeCommand]);

  const handleInputChange = useCallback((e) => {
    setState(prev => ({
      ...prev,
      input: e.target.value
    }));
  }, []);
  const handleClearTerminal = useCallback(() => {
    setState(prev => ({
      ...prev,
      output: [''],
      input: ''
    }));
  }, []);

  const handleStatusCommand = useCallback(() => {
    setState(prev => ({
      ...prev,
      input: 'status'
    }));
  }, []);
  const renderHeader = () => (
    <header className="main-header">
      <div className="main-header-left">
        <div className="main-logo"></div>
        <div>
          <h1 className="main-title">ExtreamFS</h1>
          <p className="main-subtitle">Sistema de Archivos Avanzado</p>
        </div>
      </div>
      <div className="main-header-right">
        {!usuario || usuario === 'guest' ? (
          <button onClick={handleLogin} className="main-login-btn">
            Iniciar Sesión
          </button>
        ) : (
          <>
            <span className="main-user">{usuario}</span>
            <button onClick={handleExplorer} className="main-explorer-btn">
              Explorador
            </button>
            <button onClick={handleLogout} className="main-logout-btn">
              Cerrar Sesión
            </button>
          </>
        )}
      </div>
    </header>
  );

  const renderTerminalLine = (line, idx) => (
    <div key={idx} className="terminal-line" style={{ animationDelay: `${idx * 0.1}s` }}>
      {line.startsWith('$') && (
        <span className="terminal-prompt">{usuario}@extreamfs:~</span>
      )}
      <span className={line.startsWith('$') ? 'terminal-command' : 'terminal-output'}>
        {line}
      </span>
    </div>
  );

  const renderTerminalContent = () => (
    <>
      {state.output.map(renderTerminalLine)}
      {state.isExecuting && (
        <div className="terminal-processing">
          <span className="terminal-prompt">{usuario}@extreamfs:~</span>
          <span className="cursor-blink">procesando...</span>
        </div>
      )}
      <div className="terminal-cursor">
        <span className="terminal-prompt">{usuario}@extreamfs:~</span>
        <span className="cursor-blink">█</span>
      </div>
    </>
  );

  const renderTerminalSection = () => (
    <div className="terminal-section">
      <div className="terminal-label">Terminal</div>
      
      <div ref={refs.terminal} className="terminal-box">
        {renderTerminalContent()}
      </div>

      <div className="terminal-input-area">
        <textarea
          ref={refs.input}
          value={state.input}
          onChange={handleInputChange}
          onKeyDown={handleKeyPress}
          placeholder="Escribe tu comando aquí..."
          className="terminal-input"
          disabled={state.isExecuting}
          rows="1"
        />
        <button
          onClick={executeCommand}
          disabled={state.isExecuting || !state.input.trim()}
          className="main-execute-btn"
        >
          {state.isExecuting ? 'Ejecutando...' : 'Ejecutar'}
        </button>
      </div>
    </div>
  );

  const renderQuickActions = () => (
    <div className="quick-actions">
      <input
        type="file"
        ref={refs.fileInput}
        onChange={handleFileChange}
        style={{ display: 'none' }}
        accept=".smia,.txt,.sh,.conf"
      />
      <button onClick={handleClearTerminal} className="quick-action-btn">
        Limpiar Terminal
      </button>
      <button onClick={handleChooseFileClick} className="quick-action-btn">
        Cargar Archivo
      </button>
    </div>
  );

  const renderLoadingOverlay = () => (
    state.loading && (
      <div className="main-loading-overlay">
        <div className="main-spinner"></div>
        <span className="main-loading-text">Cargando...</span>
      </div>
    )
  );
  return (
    <div className="main-bg">
      {renderLoadingOverlay()}
      {renderHeader()}
      
      <main className="main-content">
        {renderTerminalSection()}
        {renderQuickActions()}
      </main>
    </div>
  );
};

export default Main;