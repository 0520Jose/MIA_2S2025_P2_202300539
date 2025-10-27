import React, { useState, useRef, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import Cookies from 'js-cookie';
import './Main.css';

const Main = () => {
  const [state, setState] = useState({
    input: '',
    output: [''],
    isExecuting: false,
    loading: false,
    commandHistory: [],
    historyIndex: -1,
  });

  const [modal, setModal] = useState({ 
    open: false, 
    content: '', 
    title: '',
    type: 'text',
    operations: [],
    journalData: []
  });

  const refs = {
    fileInput: useRef(null),
    terminal: useRef(null),
    input: useRef(null)
  };
  const navigate = useNavigate();
  const [usuario, setUsuario] = useState(Cookies.get('usuario') || 'guest');
  
  useEffect(() => {
    const currentUser = Cookies.get('usuario') || 'guest';
    setUsuario(currentUser);
  }, []);

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

  useEffect(() => {
    const savedHistory = localStorage.getItem('extreamfs_command_history');
    if (savedHistory) {
      try {
        const history = JSON.parse(savedHistory);
        setState(prev => ({
          ...prev,
          commandHistory: history
        }));
      } catch (error) {
        console.error('Error loading command history:', error);
      }
    }
  }, []);

  useEffect(() => {
    if (state.commandHistory.length > 0) {
      localStorage.setItem('extreamfs_command_history', JSON.stringify(state.commandHistory));
    }
  }, [state.commandHistory]);

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
  
  const handleLogout = useCallback(async () => {
    setState(prev => ({ ...prev, loading: true }));
    
    try {
      const logoutCommand = `logout`;
      
      const response = await fetch('http://ec2-18-223-185-41.us-east-2.compute.amazonaws.com:8000/execute', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ 
          comando: logoutCommand 
        }),
      });

      const data = await response.json();
      const output = data.salida || '';
      
      if (!output.toLowerCase().includes('error')) {
        Cookies.remove('usuario');
        setUsuario('guest');
        setState(prev => ({ 
          ...prev, 
          loading: false,
          isExecuting: false,
          
          output: [...prev.output, `$ ${logoutCommand}`, output]
        }));
      } else {
        setState(prev => ({ 
          ...prev, 
          loading: false,
          isExecuting: false,
          output: [...prev.output, `$ ${logoutCommand}`, output]
        }));
      }

    } catch (error) {
      console.error('Error en logout:', error);
      setState(prev => ({ 
        ...prev, 
        loading: false,
        isExecuting: false,
        output: [...prev.output, `Error: ${error.message}`]
      }));
    }
  }, []);

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

  const addToHistory = useCallback((command) => {
    if (!command.trim()) return;
    
    setState(prev => {
      const newHistory = [...prev.commandHistory];
      
      if (newHistory[newHistory.length - 1] !== command) {
        newHistory.push(command);
        
        if (newHistory.length > 100) {
          newHistory.shift();
        }
      }
      
      return {
        ...prev,
        commandHistory: newHistory,
        historyIndex: -1
      };
    });
  }, []);

  const navigateHistory = useCallback((direction) => {
    setState(prev => {
      if (prev.commandHistory.length === 0) return prev;
      
      let newIndex = prev.historyIndex;
      
      if (direction === 'up') {
        newIndex = newIndex < prev.commandHistory.length - 1 ? newIndex + 1 : prev.commandHistory.length - 1;
      } else {
        newIndex = newIndex > 0 ? newIndex - 1 : -1;
      }
      
      const newInput = newIndex >= 0 ? prev.commandHistory[prev.commandHistory.length - 1 - newIndex] : '';
      
      return {
        ...prev,
        historyIndex: newIndex,
        input: newInput
      };
    });
  }, []);

  const clearHistory = useCallback(() => {
    setState(prev => ({
      ...prev,
      commandHistory: [],
      historyIndex: -1,
      input: ''
    }));
    localStorage.removeItem('extreamfs_command_history');
  }, []);

  const parseJournalingJSON = (content) => {
    try {
      const jsonData = JSON.parse(content);
      if (jsonData.entries && Array.isArray(jsonData.entries)) {
        return jsonData.entries;
      }
      return [];
    } catch (error) {
      return parseJournalingContent(content);
    }
  };

  const parseJournalingContent = (content) => {
    const lines = content.split('\n');
    const operations = [];
    let currentOperation = null;
    
    lines.forEach(line => {
      const trimmed = line.trim();
      if (!trimmed) return;
      
      if (trimmed.startsWith('Recuperando operación:') || trimmed.startsWith('Operación')) {
        if (currentOperation) {
          operations.push(currentOperation);
        }
        
        const operationIdMatch = trimmed.match(/(\d+)/);
        currentOperation = {
          id: operationIdMatch ? parseInt(operationIdMatch[1]) : operations.length + 1,
          type: 'operation',
          description: trimmed,
          status: 'processing',
          details: [],
          timestamp: new Date().toLocaleTimeString()
        };
      } 
      else if (trimmed.startsWith('Error')) {
        if (currentOperation) {
          currentOperation.status = 'error';
          currentOperation.details.push({
            type: 'error',
            content: trimmed
          });
        }
      }
      else if (trimmed.includes('creado') || trimmed.includes('existe') || trimmed.includes('exitosamente')) {
        if (currentOperation) {
          currentOperation.status = 'success';
          currentOperation.details.push({
            type: 'success',
            content: trimmed
          });
        } else {
          operations.push({
            id: operations.length + 1,
            type: 'info',
            description: trimmed,
            status: 'success',
            details: [],
            timestamp: new Date().toLocaleTimeString()
          });
        }
      }
      else if (trimmed.startsWith('Simulando') || trimmed.startsWith('Verificación') || trimmed.startsWith('Inodo') || trimmed.includes('informativo')) {
        if (currentOperation) {
          currentOperation.details.push({
            type: 'info',
            content: trimmed
          });
        } else {
          operations.push({
            id: operations.length + 1,
            type: 'info',
            description: trimmed,
            status: 'info',
            details: [],
            timestamp: new Date().toLocaleTimeString()
          });
        }
      }
      else if (currentOperation) {
        currentOperation.details.push({
          type: 'normal',
          content: trimmed
        });
      }
    });
    
    if (currentOperation) {
      operations.push(currentOperation);
    }
    
    return operations;
  };

  const executeCommand = useCallback(async () => {
    if (!state.input.trim()) return;

    const command = state.input.trim();
    
    addToHistory(command);
    
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

    if (command.toLowerCase() === 'history') {
      setTimeout(() => {
        const historyOutput = state.commandHistory.length > 0 
          ? state.commandHistory.map((cmd, idx) => `${idx + 1}  ${cmd}`).join('\n')
          : 'No hay comandos en el historial';
        
        setState(prev => ({
          ...prev,
          output: [...prev.output, historyOutput],
          isExecuting: false,
          input: ''
        }));
      }, 300);
      return;
    }

    if (command.toLowerCase() === 'clear-history') {
      setTimeout(() => {
        clearHistory();
        setState(prev => ({
          ...prev,
          output: [...prev.output, 'Historial limpiado exitosamente'],
          isExecuting: false,
          input: ''
        }));
      }, 300);
      return;
    }

    if (command.toLowerCase().startsWith('journaling -id=')) {
      try {
        const response = await fetch('http://ec2-18-223-185-41.us-east-2.compute.amazonaws.com:8000/execute', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ comando: command }),
        });

        const data = await response.json();
        const content = data.salida || 'Sin respuesta del servidor';

        setTimeout(() => {
          try {
            const jsonData = JSON.parse(content);
            if (jsonData.entries && Array.isArray(jsonData.entries)) {
              setModal({ 
                open: true, 
                content: content,
                title: 'Registro de Journaling - EXT3',
                type: 'journaling-table',
                journalData: jsonData.entries
              });
            } else {
              throw new Error('No es formato JSON esperado');
            }
          } catch (error) {
            const journalingData = parseJournalingContent(content);
            setModal({ 
              open: true, 
              content: content,
              title: 'Recuperación del Sistema EXT3 - Journal',
              type: 'journaling',
              operations: journalingData
            });
          }
          
          setState(prev => ({
            ...prev,
            isExecuting: false,
            input: ''
          }));
        }, 400);

      } catch (error) {
        setTimeout(() => {
          setModal({ 
            open: true, 
            content: `Error: ${error.message}`,
            title: 'Error en Journaling',
            type: 'text'
          });
          setState(prev => ({
            ...prev,
            isExecuting: false,
            input: ''
          }));
        }, 400);
      }
      return;
    }

    if (command.toLowerCase().includes('login')) {
      try {
      const response = await fetch('http://ec2-18-223-185-41.us-east-2.compute.amazonaws.com:8000/execute', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ comando: command }),
      });

      const data = await response.json();
      const output = data.salida || '';
      
      setTimeout(() => {
        if (!output.toLowerCase().includes('error')) {
        const userMatch = command.match(/login\s+-user=(\w+)/i);
        if (userMatch && userMatch[1]) {
          const username = userMatch[1];
          Cookies.set('usuario', username, { expires: 1 });
          setUsuario(username);
        }
        }
        
        setState(prev => ({
        ...prev,
        output: [...prev.output, output],
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
      return;
    }

    if (command.toLowerCase().includes('logout')) {
      handleLogout();
      return;
    }

    try {
      const response = await fetch('http://ec2-18-223-185-41.us-east-2.compute.amazonaws.com:8000/execute', {
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
  }, [state.input, state.commandHistory, addToHistory, clearHistory, handleLogout]);

  const handleKeyPress = useCallback((e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      executeCommand();
    }
    
    if (e.key === 'ArrowUp') {
      e.preventDefault();
      navigateHistory('up');
    }
    
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      navigateHistory('down');
    }
    
    if (e.key === 'ArrowUp' && e.ctrlKey) {
      e.preventDefault();
      setState(prev => ({
        ...prev,
        input: 'status'
      }));
    }

    if (e.key === 'Escape') {
      e.preventDefault();
      setState(prev => ({
        ...prev,
        input: '',
        historyIndex: -1
      }));
    }
  }, [executeCommand, navigateHistory]);

  const handleInputChange = useCallback((e) => {
    setState(prev => ({
      ...prev,
      input: e.target.value,
      historyIndex: -1
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

  const showHistoryModal = useCallback(() => {
    setModal({
      open: true,
      title: 'Historial de Comandos',
      content: state.commandHistory.length > 0 
        ? state.commandHistory.map((cmd, idx) => `${idx + 1}. ${cmd}`).join('\n')
        : 'No hay comandos en el historial',
      type: 'text'
    });
  }, [state.commandHistory]);

  const renderJournalingTableModal = () => {
    if (!modal.open || modal.type !== 'journaling-table') return null;

    const journalEntries = modal.journalData || [];

    return (
      <div className="main-modal-overlay" onClick={() => setModal({ open: false, content: '', title: '', type: 'text' })}>
        <div className="main-modal journaling-table-modal" onClick={(e) => e.stopPropagation()}>
          <div className="main-modal-header">
            <h2 className="main-modal-title">{modal.title}</h2>
            <button 
              className="main-modal-close-x" 
              onClick={() => setModal({ open: false, content: '', title: '', type: 'text' })}
            >
              ✕
            </button>
          </div>
          
          <div className="journaling-table-content">
            <div className="journaling-stats">
              <div className="stat-item">
                <span className="stat-label">Total de Entradas:</span>
                <span className="stat-value">{journalEntries.length}</span>
              </div>
              <div className="stat-item">
                <span className="stat-label">Operaciones Únicas:</span>
                <span className="stat-value">
                  {new Set(journalEntries.map(entry => entry.operation)).size}
                </span>
              </div>
              <div className="stat-item">
                <span className="stat-label">Período:</span>
                <span className="stat-value">
                  {journalEntries.length > 0 ? 
                    `${journalEntries[0].date} - ${journalEntries[journalEntries.length - 1].date}` 
                    : 'N/A'}
                </span>
              </div>
            </div>

            <div className="journaling-table-container">
              <table className="journaling-table">
                <thead>
                  <tr>
                    <th className="col-operation">Operación</th>
                    <th className="col-path">Ruta</th>
                    <th className="col-content">Contenido</th>
                    <th className="col-date">Fecha/Hora</th>
                    <th className="col-actions">Acciones</th>
                  </tr>
                </thead>
                <tbody>
                  {journalEntries.map((entry, index) => (
                    <tr key={index} className="journal-entry">
                      <td className="col-operation">
                        <span className={`operation-badge operation-${entry.operation.toLowerCase()}`}>
                          {entry.operation}
                        </span>
                      </td>
                      <td className="col-path">
                        <div className="path-display" title={entry.path}>
                          {entry.path}
                        </div>
                      </td>
                      <td className="col-content">
                        <div className="content-preview" title={entry.content}>
                          {entry.content && entry.content.length > 50 
                            ? `${entry.content.substring(0, 50)}...` 
                            : entry.content || '-'}
                        </div>
                      </td>
                      <td className="col-date">
                        {entry.date}
                      </td>
                      <td className="col-actions">
                        <button 
                          className="table-action-btn view-details"
                          onClick={() => handleViewEntryDetails(entry)}
                          title="Ver detalles"
                        >
                          👁️
                        </button>
                        <button 
                          className="table-action-btn copy-path"
                          onClick={() => handleCopyPath(entry.path)}
                          title="Copiar ruta"
                        >
                          📋
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {journalEntries.length === 0 && (
              <div className="no-entries">
                No se encontraron entradas en el journal
              </div>
            )}
          </div>
          
          <div className="main-modal-footer">
            <button 
              className="main-modal-close" 
              onClick={() => setModal({ open: false, content: '', title: '', type: 'text' })}
            >
              Cerrar
            </button>
            <button 
              className="main-modal-export" 
              onClick={() => handleExportJournal(journalEntries)}
            >
              Exportar JSON
            </button>
          </div>
        </div>
      </div>
    );
  };

  const handleViewEntryDetails = (entry) => {
    setModal({
      open: true,
      title: `Detalles de Operación: ${entry.operation}`,
      content: JSON.stringify(entry, null, 2),
      type: 'text'
    });
  };

  const handleCopyPath = (path) => {
    navigator.clipboard.writeText(path).then(() => {
      console.log('Ruta copiada:', path);
    });
  };

  const handleExportJournal = (entries) => {
    const dataStr = JSON.stringify({ entries }, null, 2);
    const dataBlob = new Blob([dataStr], { type: 'application/json' });
    const url = URL.createObjectURL(dataBlob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `journaling-${new Date().toISOString().split('T')[0]}.json`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  const renderJournalingModal = () => {
    if (!modal.open || modal.type !== 'journaling') return null;

    return (
      <div className="main-modal-overlay" onClick={() => setModal({ open: false, content: '', title: '', type: 'text' })}>
        <div className="main-modal journaling-modal" onClick={(e) => e.stopPropagation()}>
          <div className="main-modal-header">
            <h2 className="main-modal-title">{modal.title}</h2>
            <button 
              className="main-modal-close-x" 
              onClick={() => setModal({ open: false, content: '', title: '', type: 'text' })}
            >
              ✕
            </button>
          </div>
          
          <div className="journaling-content">
            <div className="journaling-summary">
              <div className="summary-item">
                <span className="summary-label">Total de Operaciones:</span>
                <span className="summary-value">{modal.operations?.length || 0}</span>
              </div>
              <div className="summary-item">
                <span className="summary-label">Éxitos:</span>
                <span className="summary-value success">
                  {modal.operations?.filter(op => op.status === 'success').length || 0}
                </span>
              </div>
              <div className="summary-item">
                <span className="summary-label">Errores:</span>
                <span className="summary-value error">
                  {modal.operations?.filter(op => op.status === 'error').length || 0}
                </span>
              </div>
            </div>

            <div className="journaling-operations">
              {modal.operations?.map((operation, idx) => (
                <div key={idx} className={`operation-card ${operation.status}`}>
                  <div className="operation-header">
                    <div className="operation-icon">
                      {operation.status === 'success' && '✓'}
                      {operation.status === 'error' && '✗'}
                      {operation.status === 'processing' && '⟳'}
                      {operation.status === 'info' && 'ℹ'}
                    </div>
                    <div className="operation-info">
                      <div className="operation-title">
                        Operación #{operation.id}
                      </div>
                      <div className="operation-description">
                        {operation.description}
                      </div>
                    </div>
                    <div className="operation-timestamp">
                      {operation.timestamp}
                    </div>
                  </div>
                  
                  {operation.details.length > 0 && (
                    <div className="operation-details">
                      {operation.details.map((detail, detailIdx) => (
                        <div key={detailIdx} className={`detail-item ${detail.type}`}>
                          <span className="detail-icon">
                            {detail.type === 'success' && '✓'}
                            {detail.type === 'error' && '✗'}
                            {detail.type === 'info' && 'ℹ'}
                            {detail.type === 'normal' && '•'}
                          </span>
                          <span className="detail-content">{detail.content}</span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
          
          <div className="main-modal-footer">
            <button 
              className="main-modal-close" 
              onClick={() => setModal({ open: false, content: '', title: '', type: 'text' })}
            >
              Cerrar
            </button>
          </div>
        </div>
      </div>
    );
  };

  const renderTextModal = () => {
    if (!modal.open || modal.type !== 'text') return null;

    return (
      <div className="main-modal-overlay" onClick={() => setModal({ open: false, content: '', title: '', type: 'text' })}>
        <div className="main-modal" onClick={(e) => e.stopPropagation()}>
          <div className="main-modal-header">
            <h2 className="main-modal-title">{modal.title}</h2>
            <button 
              className="main-modal-close-x" 
              onClick={() => setModal({ open: false, content: '', title: '', type: 'text' })}
            >
              ✕
            </button>
          </div>
          
          <div className="main-modal-content">
            <pre className="modal-text-content">{modal.content}</pre>
          </div>
          
          <div className="main-modal-footer">
            <button 
              className="main-modal-close" 
              onClick={() => setModal({ open: false, content: '', title: '', type: 'text' })}
            >
              Cerrar
            </button>
          </div>
        </div>
      </div>
    );
  };
  
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
      <div className="terminal-label">
        Terminal
        <span className="history-info">
          ({state.commandHistory.length} comandos en historial)
        </span>
      </div>
      
      <div ref={refs.terminal} className="terminal-box">
        {renderTerminalContent()}
      </div>

      <div className="terminal-input-area">
        <textarea
          ref={refs.input}
          value={state.input}
          onChange={handleInputChange}
          onKeyDown={handleKeyPress}
          placeholder="Escribe tu comando aquí... (↑↓ para navegar historial, Esc para limpiar)"
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
      <button onClick={showHistoryModal} className="quick-action-btn">
        Ver Historial
      </button>
      <button onClick={clearHistory} className="quick-action-btn danger">
        Limpiar Historial
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
      {renderJournalingTableModal()}
      {renderJournalingModal()}
      {renderTextModal()}
      {renderHeader()}
      <main className="main-content">
        {renderTerminalSection()}
        {renderQuickActions()}
      </main>
    </div>
  );
};

export default Main;