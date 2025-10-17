import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { HardDrive, Folder, File, ArrowLeft, Home, Monitor } from 'lucide-react';
import './Explorer.css';

const FileExplorer = () => {
  const navigate = useNavigate();

  const [disks, setDisks] = useState([]);
  const [selectedDisk, setSelectedDisk] = useState(null);
  const [partitions, setPartitions] = useState([]);
  const [selectedPartition, setSelectedPartition] = useState(null);
  const [currentPath, setCurrentPath] = useState('/');
  const [files, setFiles] = useState([]);
  const [selectedFile, setSelectedFile] = useState(null);
  const [fileContent, setFileContent] = useState('');

  const [loadingDisks, setLoadingDisks] = useState(true);
  const [loadingPartitions, setLoadingPartitions] = useState(false);
  const [loadingFiles, setLoadingFiles] = useState(false);

  const formatSize = (bytes) => {
    if (bytes < 1024) return `${bytes} B`;
    const kb = bytes / 1024;
    if (kb < 1024) return `${kb.toFixed(2)} KB`;
    const mb = kb / 1024;
    if (mb < 1024) return `${mb.toFixed(2)} MB`;
    const gb = mb / 1024;
    return `${gb.toFixed(2)} GB`;
  };

  useEffect(() => {
    setLoadingDisks(true);
    fetch('/api/disks')
      .then(res => res.json())
      .then(data => {
        setDisks(Array.isArray(data) ? data : []);
      })
      .catch(() => setDisks([]))
      .finally(() => setLoadingDisks(false));
  }, []);

  useEffect(() => {
    if (selectedDisk) {
      setLoadingPartitions(true);
      fetch(`/api/partitions?path=${encodeURIComponent(selectedDisk.path)}`)
        .then(res => res.json())
        .then(data => {
          setPartitions(Array.isArray(data) ? data : []);
        })
        .catch(() => setPartitions([]))
        .finally(() => setLoadingPartitions(false));
    } else {
      setPartitions([]);
      setLoadingPartitions(false);
    }
  }, [selectedDisk]);

  useEffect(() => {
    if (selectedDisk && selectedPartition) {
      setLoadingFiles(true);
      fetch(`/api/files?disk=${encodeURIComponent(selectedDisk.path)}&partition=${encodeURIComponent(selectedPartition.name)}&path=${encodeURIComponent(currentPath)}`)
        .then(res => res.json())
        .then(data => setFiles(Array.isArray(data) ? data : []))
        .catch(() => setFiles([]))
        .finally(() => setLoadingFiles(false));
    } else {
      setFiles([]);
      setLoadingFiles(false);
    }
  }, [selectedDisk, selectedPartition, currentPath]);

  useEffect(() => {
    if (selectedFile && selectedDisk && selectedPartition) {
      const filePath = currentPath.endsWith('/') ? currentPath + selectedFile : `${currentPath}/${selectedFile}`;
      fetch(`/api/file-content?disk=${encodeURIComponent(selectedDisk.path)}&partition=${encodeURIComponent(selectedPartition.name)}&path=${encodeURIComponent(filePath)}`)
        .then(res => res.json())
        .then(data => setFileContent(data.content || ''))
        .catch(() => setFileContent('Error al cargar archivo'));
    }
  }, [selectedFile]);

  const goToMain = () => {
    navigate('/main');
  };

  if (!selectedDisk) {
    return (
      <div className="ext-explorer-container">
        <div className="ext-container">
          <div className="ext-header">
            <div className="ext-icon-wrapper">
              <HardDrive className="ext-main-icon" />
            </div>
            <div>
              <h1 className="ext-title">Visualizador del sistema de archivos</h1>
              <p className="ext-subtitle">Explorador de sistemas de archivos Linux EXT2 y EXT3</p>
            </div>
            <button onClick={goToMain} className="ext-button ext-button-secondary" title="Volver al inicio">
                Inicio
              </button>
          </div>
          {loadingDisks ? (
            <div className="ext-loading">Cargando discos...</div>
          ) : disks.length === 0 ? (
            <div className="ext-empty-state">No se encontraron discos.</div>
          ) : (
            <div className="ext-disk-grid">
              {disks.map(disk => (
                <div key={disk.path} className="ext-disk-card">
                  <div className="ext-card-content">
                    <div className="ext-card-header">
                      <div className="ext-disk-info">
                        <div className="ext-disk-icon ext-hdd">
                          <HardDrive className="ext-icon" />
                        </div>
                        <div>
                          <h3 className="ext-disk-name">{disk.name}</h3>
                          <p className="ext-disk-details">{disk.path}</p>
                          <p className="ext-disk-meta">
                            Capacidad: {formatSize(disk.size)} <br />
                            Fit: {disk.fit} <br />
                            Particiones: {disk.partitions?.length || 0}
                          </p>
                        </div>
                      </div>
                      <span className="ext-linux-badge">Linux</span>
                    </div>
                    <div className="ext-card-footer">
                      <button onClick={() => setSelectedDisk(disk)} className="ext-button ext-button-primary">
                        Explorar
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    );
  }

  if (!selectedPartition) {
    return (
      <div className="ext-explorer-container">
        <div className="ext-container">
          <div className="ext-nav-header">
            <button onClick={() => setSelectedDisk(null)} className="ext-back-button">
              <ArrowLeft className="ext-back-icon" />
            </button>
            <div>
              <h1 className="ext-page-title">Particiones de {selectedDisk.name}</h1>
              <p className="ext-page-subtitle">Selecciona una partición EXT2/EXT3 para explorar su contenido</p>
            </div>
            <button onClick={goToMain} className="ext-button ext-button-secondary" title="Volver al inicio">
              Inicio
            </button>
          </div>
          {loadingPartitions ? (
            <div className="ext-loading">Cargando particiones...</div>
          ) : partitions.length === 0 ? (
            <div className="ext-empty-state">No se encontraron particiones EXT2/EXT3 en este disco.</div>
          ) : (
            <div className="ext-partition-grid">
              {partitions.map(partition => (
                <div key={partition.name} className="ext-partition-card">
                  <div className="ext-card-content">
                    <div className="ext-partition-header">
                      <div className="ext-partition-info">
                        <div className={`ext-partition-icon`}>
                          <Monitor className="ext-icon" />
                        </div>
                        <div>
                          <h3 className="ext-partition-name">{partition.name}</h3>
                          <p className="ext-partition-details">{partition.type}</p>
                          <p className="ext-partition-meta">
                            Tamaño: {formatSize(partition.size)} <br />
                            Fit: {partition.fit} <br />
                            Estado: {partition.status}
                          </p>
                        </div>
                      </div>
                      <span className={`ext-status-badge ${partition.status}`}>
                        {partition.status}
                      </span>
                    </div>
                    <button
                      onClick={() => {
                        setSelectedPartition(partition);
                        setCurrentPath('/');
                      }}
                      className="ext-button ext-button-primary ext-button-full"
                    >
                      Explorar Archivos
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    );
  }

  if (selectedFile) {
    return (
      <div className="ext-explorer-container">
        <div className="ext-container">
          <div className="ext-nav-header">
            <button onClick={() => setSelectedFile(null)} className="ext-back-button">
              <ArrowLeft className="ext-back-icon" />
            </button>
            <div className="ext-file-header">
              <div className="ext-file-icon-wrapper">
                <File className="ext-file-icon" />
              </div>
              <div>
                <h1 className="ext-page-title">{selectedFile}</h1>
              </div>
            </div>
            <button onClick={goToMain} className="ext-button ext-button-secondary" title="Volver al inicio">
              Inicio
            </button>
          </div>
          <div className="ext-file-container">
            <div className="ext-file-card">
              <div className="ext-file-card-header">
                <h3 className="ext-file-card-title">Contenido del archivo</h3>
              </div>
              <div className="ext-file-card-content">
                <pre className="ext-file-content">{fileContent}</pre>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="ext-explorer-container">
      <div className="ext-container">
        <div className="ext-explorer-header">
          <div className="ext-nav-header">
            <button onClick={() => setSelectedPartition(null)} className="ext-back-button">
              <ArrowLeft className="ext-back-icon" />
            </button>
            <div>
              <h1 className="ext-page-title">Explorador de Archivos</h1>
              <div className="ext-breadcrumb">
                <span className="ext-breadcrumb-current">/</span>
                {currentPath !== '/' && (
                  <>
                    {currentPath.split('/').filter(Boolean).map((part, idx, arr) => (
                      <React.Fragment key={idx}>
                        <span className="ext-breadcrumb-separator">/</span>
                        <span className={idx === arr.length - 1 ? 'ext-breadcrumb-current' : 'ext-breadcrumb-item'}>
                          {part}
                        </span>
                      </React.Fragment>
                    ))}
                  </>
                )}
              </div>
            </div>
            <div className="ext-header-actions">
              {currentPath !== '/' && (
                <button
                  onClick={() => {
                    const parts = currentPath.split('/').filter(Boolean);
                    parts.pop();
                    setCurrentPath(parts.length ? '/' + parts.join('/') : '/');
                    setSelectedFile(null);
                    setFileContent('');
                  }}
                  className="ext-button ext-button-secondary"
                >
                  Subir
                </button>
              )}
              <button onClick={goToMain} className="ext-button ext-button-secondary" title="Volver al inicio">
                Inicio
              </button>
            </div>
          </div>
        </div>
        <div className="ext-file-explorer">
          <div className="ext-file-explorer-card">
            <div className="ext-file-explorer-header">
              <div className="ext-partition-indicator">
                <Home className="ext-home-icon" />
                <span className="ext-partition-name">{selectedPartition.name}</span>
              </div>
            </div>
            <div className="ext-file-explorer-content">
              {loadingFiles ? (
                <div className="ext-loading">Cargando archivos...</div>
              ) : files.length === 0 ? (
                <div className="ext-empty-folder">
                  <div className="ext-empty-folder-icon">
                    <Folder className="ext-empty-icon" />
                  </div>
                  <h3 className="ext-empty-folder-title">Directorio vacío</h3>
                  <p className="ext-empty-folder-subtitle">
                    No hay archivos o directorios en esta ubicación
                  </p>
                </div>
              ) : (
                <div className="ext-file-grid">
                  {files.map(node => (
                    <div
                      key={node.name}
                      className="ext-file-item"
                      onClick={() => {
                        if (node.type === 'd') {
                          setCurrentPath(currentPath === '/' ? `/${node.name}` : `${currentPath}/${node.name}`);
                          setSelectedFile(null);
                          setFileContent('');
                        } else if (node.type === 'f') {
                          setSelectedFile(node.name);
                        }
                      }}
                    >
                      <div className="ext-file-item-content">
                        <div className={`ext-file-item-icon ${node.type === 'd' ? 'ext-folder-icon' : 'ext-file-icon-item'}`}>
                          {node.type === 'd' ? <Folder className="ext-icon" /> : <File className="ext-icon" />}
                        </div>
                        <div className="ext-file-item-details">
                          <h4 className="ext-file-item-name">{node.name}</h4>
                          <p className="ext-file-item-meta">
                            {node.type === 'd'
                              ? 'Directorio'
                              : `${formatSize(node.size || 0)} • Perms: ${Array.isArray(node.permissions) ? node.permissions.join('') : ''}`}
                          </p>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default FileExplorer;