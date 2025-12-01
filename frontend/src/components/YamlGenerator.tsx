import React, { useState } from 'react';
import { FileOutput } from 'lucide-react';
import YAML from 'yaml';

// --- Interfaces ---

interface OptionsConfig {
    dcfPath: string;
    heartbeatMultiplier: number;
}

interface PdoMapping {
    index: number;
    sub_index: number;
}

interface PdoConfig {
    id: number; // Internal ID for React keys
    pdoNumber: number; // 1-512
    type: 'RPDO' | 'TPDO';
    enabled: boolean;
    cobIdType: 'auto' | 'manual';
    cobId?: number;
    transmission: number;
    inhibitTime: number; // 100us multiples
    eventTimer: number; // ms
    eventDeadline?: number; // ms (RPDO only)
    syncStart?: number; // (TPDO only)
    mapping: PdoMapping[];
}

interface SdoConfig {
    id: number;
    index: number;
    sub_index: number;
    value: number;
}

interface SlaveConfig {
    id: number;
    name: string;
    dcfPath: string;
    nodeId?: number;
    revisionNumber?: number;
    serialNumber?: number;
    heartbeatMultiplier?: number;
    heartbeatConsumer: boolean;
    heartbeatProducer: number; // ms
    boot: boolean;
    mandatory: boolean;
    resetCommunication: boolean;
    softwareFile: string;
    softwareVersion?: number;
    configurationFile: string;
    restoreConfiguration: number;
    errorBehavior: Record<number, number>; // Simplified for now, can be expanded
    rpdos: PdoConfig[];
    tpdos: PdoConfig[];
    sdos: SdoConfig[];
}

interface MasterConfig {
    nodeId: number;
    baudrate: number;
    vendorId?: number;
    productCode?: number;
    revisionNumber?: number;
    serialNumber?: number;
    syncPeriod: number; // us
    syncWindow: number; // us
    syncOverflow: number;
    heartbeatProducer: number; // ms
    heartbeatConsumer: boolean;
    emcyInhibitTime: number; // 100us
    nmtInhibitTime: number; // 100us
    start: boolean;
    startNodes: boolean;
    startAllNodes: boolean;
    resetAllNodes: boolean;
    stopAllNodes: boolean;
    bootTime: number; // ms
    errorBehavior: Record<number, number>; // Simplified
}

// --- Fieldset Helper ---
const ConfigField: React.FC<{ label: string; description?: string; children: React.ReactNode }> = ({ label, description, children }) => (
    <fieldset className="fieldset w-full bg-base-100 p-4 rounded-box border border-base-300">
        <legend className="fieldset-legend text-sm font-bold uppercase tracking-wider opacity-70">{label}</legend>
        {children}
        {description && <p className="fieldset-label text-xs opacity-60 mt-1">{description}</p>}
    </fieldset>
);

// --- Component ---

const YamlGenerator: React.FC = () => {
    // --- State ---
    const [options, setOptions] = useState<OptionsConfig>({
        dcfPath: '',
        heartbeatMultiplier: 3.0,
    });

    const [master, setMaster] = useState<MasterConfig>({
        nodeId: 255,
        baudrate: 1000,
        vendorId: 0,
        productCode: 0,
        revisionNumber: 0,
        serialNumber: 0,
        syncPeriod: 1000000,
        syncWindow: 0,
        syncOverflow: 0,
        heartbeatProducer: 0,
        heartbeatConsumer: true,
        emcyInhibitTime: 0,
        nmtInhibitTime: 0,
        start: true,
        startNodes: true,
        startAllNodes: false,
        resetAllNodes: false,
        stopAllNodes: false,
        bootTime: 0,
        errorBehavior: { 1: 0x00 },
    });

    const [slaves, setSlaves] = useState<SlaveConfig[]>([]);
    const [selection, setSelection] = useState<'options' | 'master' | { type: 'slave', id: number }>('options');
    const [showPreview, setShowPreview] = useState(false);

    // --- Helpers ---
    const updateOptions = (field: keyof OptionsConfig, value: any) => {
        setOptions(prev => ({ ...prev, [field]: value }));
    };

    const updateMaster = (field: keyof MasterConfig, value: any) => {
        setMaster(prev => ({ ...prev, [field]: value }));
    };

    const addSlave = () => {
        const newSlave: SlaveConfig = {
            id: Date.now(),
            name: `slave_${slaves.length + 1}`,
            dcfPath: 'slave.eds',
            nodeId: slaves.length + 1,
            heartbeatConsumer: false,
            heartbeatProducer: 0,
            boot: true,
            mandatory: false,
            resetCommunication: true,
            softwareFile: "",
            configurationFile: "",
            restoreConfiguration: 0,
            errorBehavior: {},
            rpdos: [],
            tpdos: [],
            sdos: []
        };
        setSlaves([...slaves, newSlave]);
        setSelection({ type: 'slave', id: newSlave.id });
    };

    const updateSlave = (id: number, field: keyof SlaveConfig, value: any) => {
        setSlaves(slaves.map(s => s.id === id ? { ...s, [field]: value } : s));
    };

    const removeSlave = (id: number) => {
        setSlaves(slaves.filter(s => s.id !== id));
        if (selection !== 'options' && selection !== 'master' && selection.id === id) {
            setSelection('master');
        }
    };

    // --- PDO/SDO Helpers ---
    const addPdo = (slaveId: number, type: 'RPDO' | 'TPDO') => {
        setSlaves(slaves.map(s => {
            if (s.id !== slaveId) return s;
            const list = type === 'RPDO' ? s.rpdos : s.tpdos;
            const newPdo: PdoConfig = {
                id: Date.now(),
                pdoNumber: list.length + 1,
                type,
                enabled: true,
                cobIdType: 'auto',
                transmission: 255,
                inhibitTime: 0,
                eventTimer: 0,
                mapping: []
            };
            return { ...s, [type === 'RPDO' ? 'rpdos' : 'tpdos']: [...list, newPdo] };
        }));
    };

    const updatePdo = (slaveId: number, type: 'RPDO' | 'TPDO', pdoId: number, field: keyof PdoConfig, value: any) => {
        setSlaves(slaves.map(s => {
            if (s.id !== slaveId) return s;
            const listKey = type === 'RPDO' ? 'rpdos' : 'tpdos';
            const list = s[listKey] as PdoConfig[];
            return {
                ...s,
                [listKey]: list.map(p => p.id === pdoId ? { ...p, [field]: value } : p)
            };
        }));
    };

    const removePdo = (slaveId: number, type: 'RPDO' | 'TPDO', pdoId: number) => {
        setSlaves(slaves.map(s => {
            if (s.id !== slaveId) return s;
            const listKey = type === 'RPDO' ? 'rpdos' : 'tpdos';
            return {
                ...s,
                [listKey]: (s[listKey] as PdoConfig[]).filter(p => p.id !== pdoId)
            };
        }));
    };

    const addMapping = (slaveId: number, type: 'RPDO' | 'TPDO', pdoId: number) => {
        setSlaves(slaves.map(s => {
            if (s.id !== slaveId) return s;
            const listKey = type === 'RPDO' ? 'rpdos' : 'tpdos';
            const list = s[listKey] as PdoConfig[];
            return {
                ...s,
                [listKey]: list.map(p => p.id === pdoId ? { ...p, mapping: [...p.mapping, { index: 0x0000, sub_index: 0x00 }] } : p)
            };
        }));
    };

    const updateMapping = (slaveId: number, type: 'RPDO' | 'TPDO', pdoId: number, mapIndex: number, field: keyof PdoMapping, value: number) => {
        setSlaves(slaves.map(s => {
            if (s.id !== slaveId) return s;
            const listKey = type === 'RPDO' ? 'rpdos' : 'tpdos';
            const list = s[listKey] as PdoConfig[];
            return {
                ...s,
                [listKey]: list.map(p => {
                    if (p.id !== pdoId) return p;
                    const newMapping = [...p.mapping];
                    newMapping[mapIndex] = { ...newMapping[mapIndex], [field]: value };
                    return { ...p, mapping: newMapping };
                })
            };
        }));
    };

    const removeMapping = (slaveId: number, type: 'RPDO' | 'TPDO', pdoId: number, mapIndex: number) => {
        setSlaves(slaves.map(s => {
            if (s.id !== slaveId) return s;
            const listKey = type === 'RPDO' ? 'rpdos' : 'tpdos';
            const list = s[listKey] as PdoConfig[];
            return {
                ...s,
                [listKey]: list.map(p => {
                    if (p.id !== pdoId) return p;
                    return { ...p, mapping: p.mapping.filter((_, i) => i !== mapIndex) };
                })
            };
        }));
    };

    const addSdo = (slaveId: number) => {
        setSlaves(slaves.map(s => {
            if (s.id !== slaveId) return s;
            const newSdo: SdoConfig = {
                id: Date.now(),
                index: 0x0000,
                sub_index: 0x00,
                value: 0
            };
            return { ...s, sdos: [...s.sdos, newSdo] };
        }));
    };

    const updateSdo = (slaveId: number, sdoId: number, field: keyof SdoConfig, value: number) => {
        setSlaves(slaves.map(s => {
            if (s.id !== slaveId) return s;
            return {
                ...s,
                sdos: s.sdos.map(sdo => sdo.id === sdoId ? { ...sdo, [field]: value } : sdo)
            };
        }));
    };

    const removeSdo = (slaveId: number, sdoId: number) => {
        setSlaves(slaves.map(s => {
            if (s.id !== slaveId) return s;
            return { ...s, sdos: s.sdos.filter(sdo => sdo.id !== sdoId) };
        }));
    };


    // --- YAML Generation ---
    const generateYaml = () => {
        const config: any = {
            options: {
                dcf_path: options.dcfPath,
                heartbeat_multiplier: options.heartbeatMultiplier
            },
            master: {
                node_id: master.nodeId,
                baudrate: master.baudrate,
                vendor_id: master.vendorId ? `0x${master.vendorId.toString(16)}` : undefined,
                product_code: master.productCode ? `0x${master.productCode.toString(16)}` : undefined,
                revision_number: master.revisionNumber ? `0x${master.revisionNumber.toString(16)}` : undefined,
                serial_number: master.serialNumber ? `0x${master.serialNumber.toString(16)}` : undefined,
                sync_period: master.syncPeriod,
                sync_window: master.syncWindow,
                sync_overflow: master.syncOverflow,
                heartbeat_producer: master.heartbeatProducer,
                heartbeat_consumer: master.heartbeatConsumer,
                emcy_inhibit_time: master.emcyInhibitTime,
                nmt_inhibit_time: master.nmtInhibitTime,
                start: master.start,
                start_nodes: master.startNodes,
                start_all_nodes: master.startAllNodes,
                reset_all_nodes: master.resetAllNodes,
                stop_all_nodes: master.stopAllNodes,
                boot_time: master.bootTime,
                error_behavior: master.errorBehavior
            }
        };

        // Remove undefined keys
        Object.keys(config.master).forEach(key => config.master[key] === undefined && delete config.master[key]);

        slaves.forEach(slave => {
            const slaveConfig: any = {
                dcf: slave.dcfPath,
                node_id: slave.nodeId,
                boot: slave.boot,
                mandatory: slave.mandatory,
                reset_communication: slave.resetCommunication,
                heartbeat_consumer: slave.heartbeatConsumer,
                heartbeat_producer: slave.heartbeatProducer,
                software_file: slave.softwareFile || undefined,
                software_version: slave.softwareVersion ? `0x${slave.softwareVersion.toString(16)}` : undefined,
                configuration_file: slave.configurationFile || undefined,
                restore_configuration: slave.restoreConfiguration ? `0x${slave.restoreConfiguration.toString(16)}` : undefined,
                error_behavior: Object.keys(slave.errorBehavior).length > 0 ? slave.errorBehavior : undefined
            };

            if (slave.revisionNumber) slaveConfig.revision_number = `0x${slave.revisionNumber.toString(16)}`;
            if (slave.serialNumber) slaveConfig.serial_number = `0x${slave.serialNumber.toString(16)}`;
            if (slave.heartbeatMultiplier) slaveConfig.heartbeat_multiplier = slave.heartbeatMultiplier;

            // Remove undefined keys
            Object.keys(slaveConfig).forEach(key => slaveConfig[key] === undefined && delete slaveConfig[key]);

            // RPDOs
            if (slave.rpdos.length > 0) {
                slaveConfig.rpdo = {};
                slave.rpdos.forEach(pdo => {
                    const pdoConfig: any = {
                        enabled: pdo.enabled,
                        cob_id: pdo.cobIdType === 'auto' ? 'auto' : `0x${pdo.cobId?.toString(16)}`,
                        transmission: pdo.transmission
                    };
                    if (pdo.mapping.length > 0) {
                        pdoConfig.mapping = pdo.mapping.map(m => ({
                            index: `0x${m.index.toString(16)}`,
                            sub_index: `0x${m.sub_index.toString(16)}`
                        }));
                    }
                    slaveConfig.rpdo[pdo.pdoNumber] = pdoConfig;
                });
            }

            // TPDOs
            if (slave.tpdos.length > 0) {
                slaveConfig.tpdo = {};
                slave.tpdos.forEach(pdo => {
                    const pdoConfig: any = {
                        enabled: pdo.enabled,
                        cob_id: pdo.cobIdType === 'auto' ? 'auto' : `0x${pdo.cobId?.toString(16)}`,
                        transmission: pdo.transmission,
                        event_timer: pdo.eventTimer,
                        sync_start: pdo.syncStart || 0
                    };
                    if (pdo.mapping.length > 0) {
                        pdoConfig.mapping = pdo.mapping.map(m => ({
                            index: `0x${m.index.toString(16)}`,
                            sub_index: `0x${m.sub_index.toString(16)}`
                        }));
                    }
                    slaveConfig.tpdo[pdo.pdoNumber] = pdoConfig;
                });
            }

            // SDOs
            if (slave.sdos.length > 0) {
                slaveConfig.sdo = slave.sdos.map(sdo => ({
                    index: `0x${sdo.index.toString(16)}`,
                    sub_index: `0x${sdo.sub_index.toString(16)}`,
                    value: sdo.value
                }));
            }

            config[slave.name] = slaveConfig;
        });

        return YAML.stringify(config);
    };

    const copyToClipboard = () => {
        navigator.clipboard.writeText(generateYaml());
    };

    const downloadYaml = () => {
        const element = document.createElement("a");
        const file = new Blob([generateYaml()], { type: 'text/yaml' });
        element.href = URL.createObjectURL(file);
        element.download = "master.yaml";
        document.body.appendChild(element);
        element.click();
    };

    return (
        <div className="flex flex-col h-full bg-base-200">
            {/* Header */}
            <div className="bg-base-100 border-b border-base-300 p-4 flex justify-between items-center shadow-sm z-10">
                <h1 className="text-xl font-bold text-primary flex items-center gap-2">
                    <FileOutput className="h-6 w-6" />
                    YAML Generator
                </h1>
                <div className="flex gap-2">
                    <button className="btn btn-sm btn-ghost" onClick={() => setShowPreview(true)}>View YAML</button>
                    <button className="btn btn-sm btn-primary" onClick={downloadYaml}>Download</button>
                </div>
            </div>

            {/* Main Content */}
            <div className="flex-1 flex overflow-hidden">
                {/* Left Pane: Navigation */}
                <div className="w-80 bg-base-100 border-r border-base-300 flex flex-col overflow-y-auto">
                    <ul className="menu w-full p-2">
                        <li>
                            <button
                                className={selection === 'options' ? 'active' : ''}
                                onClick={() => setSelection('options')}
                            >
                                <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
                                </svg>
                                Global Options
                            </button>
                        </li>
                        <li>
                            <button
                                className={selection === 'master' ? 'active' : ''}
                                onClick={() => setSelection('master')}
                            >
                                <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                                </svg>
                                Master Configuration
                            </button>
                        </li>

                        <li className="menu-title mt-4 flex flex-row justify-between items-center">
                            <span>Slaves ({slaves.length})</span>
                            <button className="btn btn-xs btn-ghost btn-circle" onClick={addSlave} title="Add Slave">+</button>
                        </li>
                        {slaves.map(slave => (
                            <li key={slave.id}>
                                <button
                                    className={selection !== 'options' && selection !== 'master' && selection.id === slave.id ? 'active' : ''}
                                    onClick={() => setSelection({ type: 'slave', id: slave.id })}
                                >
                                    <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
                                    </svg>
                                    {slave.name} <span className="opacity-50 text-xs">(ID: {slave.nodeId})</span>
                                </button>
                            </li>
                        ))}
                    </ul>
                </div>

                {/* Right Pane: Configuration Form */}
                <div className="flex-1 bg-base-200 p-8 overflow-y-auto">
                    <div className="max-w-4xl mx-auto">
                        {selection === 'options' ? (
                            <div>
                                <h2 className="text-2xl font-bold text-primary mb-6 border-b border-base-300 pb-2">Global Options</h2>
                                <div className="bg-base-100 p-6 rounded-lg shadow-sm grid grid-cols-1 md:grid-cols-2 gap-6">
                                    <ConfigField label="DCF Path" description="The directory in which the generated .dcf and .bin files will be available at runtime (default: &quot;&quot;).">
                                        <input type="text" className="input input-bordered w-full" placeholder="/path/to/dcf/files" value={options.dcfPath} onChange={e => updateOptions('dcfPath', e.target.value)} />
                                    </ConfigField>
                                    <ConfigField label="Heartbeat Multiplier" description="The multiplication factor used to obtain heartbeat consumer times from heartbeat producer times (default: 3.0).">
                                        <input type="number" step="0.1" className="input input-bordered w-full" value={options.heartbeatMultiplier} onChange={e => updateOptions('heartbeatMultiplier', parseFloat(e.target.value))} />
                                    </ConfigField>
                                </div>
                            </div>
                        ) : selection === 'master' ? (
                            <div>
                                <h2 className="text-2xl font-bold text-primary mb-6 border-b border-base-300 pb-2">Master Configuration</h2>
                                <div className="bg-base-100 p-6 rounded-lg shadow-sm">
                                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                                        <ConfigField label="Node ID" description="The node-ID (default: 255).">
                                            <input type="number" className="input input-bordered w-full" value={master.nodeId} onChange={e => updateMaster('nodeId', parseInt(e.target.value))} />
                                        </ConfigField>
                                        <ConfigField label="Baudrate (kbit/s)" description="The baudrate in kbit/s (default: 1000).">
                                            <select className="select select-bordered w-full" value={master.baudrate} onChange={e => updateMaster('baudrate', parseInt(e.target.value))}>
                                                <option value={1000}>1000 (1M)</option>
                                                <option value={500}>500 (500k)</option>
                                                <option value={250}>250 (250k)</option>
                                                <option value={125}>125 (125k)</option>
                                            </select>
                                        </ConfigField>
                                        <ConfigField label="Vendor ID (Hex)" description="The vendor-ID (default: 0x00000000).">
                                            <input type="text" className="input input-bordered w-full" placeholder="0x..." value={master.vendorId ? master.vendorId.toString(16) : ''} onChange={e => updateMaster('vendorId', parseInt(e.target.value, 16))} />
                                        </ConfigField>
                                        <ConfigField label="Product Code (Hex)" description="The product code (default: 0x00000000).">
                                            <input type="text" className="input input-bordered w-full" placeholder="0x..." value={master.productCode ? master.productCode.toString(16) : ''} onChange={e => updateMaster('productCode', parseInt(e.target.value, 16))} />
                                        </ConfigField>
                                        <ConfigField label="Revision Number (Hex)" description="The revision number (default: 0x00000000).">
                                            <input type="text" className="input input-bordered w-full" placeholder="0x..." value={master.revisionNumber ? master.revisionNumber.toString(16) : ''} onChange={e => updateMaster('revisionNumber', parseInt(e.target.value, 16))} />
                                        </ConfigField>
                                        <ConfigField label="Serial Number (Hex)" description="The serial number (default: 0x00000000).">
                                            <input type="text" className="input input-bordered w-full" placeholder="0x..." value={master.serialNumber ? master.serialNumber.toString(16) : ''} onChange={e => updateMaster('serialNumber', parseInt(e.target.value, 16))} />
                                        </ConfigField>
                                        <ConfigField label="Sync Period (us)" description="The SYNC interval in μs (default: 0).">
                                            <input type="number" className="input input-bordered w-full" value={master.syncPeriod} onChange={e => updateMaster('syncPeriod', parseInt(e.target.value))} />
                                        </ConfigField>
                                        <ConfigField label="Sync Window (us)" description="The SYNC window length in μs (default: 0, see object 1007).">
                                            <input type="number" className="input input-bordered w-full" value={master.syncWindow} onChange={e => updateMaster('syncWindow', parseInt(e.target.value))} />
                                        </ConfigField>
                                        <ConfigField label="Sync Overflow" description="The SYNC counter overflow value (default: 0, see object 1019).">
                                            <input type="number" className="input input-bordered w-full" value={master.syncOverflow} onChange={e => updateMaster('syncOverflow', parseInt(e.target.value))} />
                                        </ConfigField>
                                        <ConfigField label="Heartbeat Producer (ms)" description="The heartbeat producer time in ms (default: 0).">
                                            <input type="number" className="input input-bordered w-full" value={master.heartbeatProducer} onChange={e => updateMaster('heartbeatProducer', parseInt(e.target.value))} />
                                        </ConfigField>
                                        <ConfigField label="Boot Time (ms)" description="The timeout for booting mandatory slaves in ms (default: 0, see object 1F89).">
                                            <input type="number" className="input input-bordered w-full" value={master.bootTime} onChange={e => updateMaster('bootTime', parseInt(e.target.value))} />
                                        </ConfigField>
                                        <ConfigField label="EMCY Inhibit Time (100us)" description="The EMCY inhibit time in multiples of 100 μs (default: 0, see object 1015).">
                                            <input type="number" className="input input-bordered w-full" value={master.emcyInhibitTime} onChange={e => updateMaster('emcyInhibitTime', parseInt(e.target.value))} />
                                        </ConfigField>
                                        <ConfigField label="NMT Inhibit Time (100us)" description="The NMT inhibit time in multiples of 100 μs (default: 0, see object 102A).">
                                            <input type="number" className="input input-bordered w-full" value={master.nmtInhibitTime} onChange={e => updateMaster('nmtInhibitTime', parseInt(e.target.value))} />
                                        </ConfigField>
                                        <ConfigField label="Heartbeat Consumer" description="Specifies whether the master should monitor the heartbeats of the slaves (default: true).">
                                            <input type="checkbox" className="checkbox checkbox-primary" checked={master.heartbeatConsumer} onChange={e => updateMaster('heartbeatConsumer', e.target.checked)} />
                                        </ConfigField>
                                    </div>

                                    <div className="divider font-bold text-secondary mt-8">NMT Flags</div>

                                    <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                                        <ConfigField label="Start (Self)" description="Specifies whether the master shall switch into the NMT operational state by itself (default: true, see bit 2 in object 1F80).">
                                            <input type="checkbox" className="checkbox checkbox-primary" checked={master.start} onChange={e => updateMaster('start', e.target.checked)} />
                                        </ConfigField>
                                        <ConfigField label="Start Nodes" description="Specifies whether the master shall start the slaves (default: true, see bit 3 in object 1F80).">
                                            <input type="checkbox" className="checkbox checkbox-primary" checked={master.startNodes} onChange={e => updateMaster('startNodes', e.target.checked)} />
                                        </ConfigField>
                                        <ConfigField label="Start All Nodes" description="Specifies whether the master shall start all nodes simultaneously (default: false, see bit 1 in object 1F80).">
                                            <input type="checkbox" className="checkbox checkbox-primary" checked={master.startAllNodes} onChange={e => updateMaster('startAllNodes', e.target.checked)} />
                                        </ConfigField>
                                        <ConfigField label="Reset All Nodes" description="Specifies whether all slaves shall be reset in case of an error event on a mandatory slave (default: false, see bit 4 in object 1F80).">
                                            <input type="checkbox" className="checkbox checkbox-primary" checked={master.resetAllNodes} onChange={e => updateMaster('resetAllNodes', e.target.checked)} />
                                        </ConfigField>
                                        <ConfigField label="Stop All Nodes" description="Specifies whether all slaves shall be stopped in case of an error event on a mandatory slave (default: false, see bit 6 in object 1F80).">
                                            <input type="checkbox" className="checkbox checkbox-primary" checked={master.stopAllNodes} onChange={e => updateMaster('stopAllNodes', e.target.checked)} />
                                        </ConfigField>
                                    </div>
                                </div>
                            </div>
                        ) : (
                            (() => {
                                const slave = slaves.find(s => s.id === selection.id);
                                if (!slave) return <div>Slave not found</div>;
                                return (
                                    <div>
                                        <div className="flex justify-between items-center mb-6 border-b border-base-300 pb-2">
                                            <h2 className="text-2xl font-bold text-primary">Slave Configuration</h2>
                                            <button className="btn btn-error btn-sm" onClick={() => removeSlave(slave.id)}>Delete Slave</button>
                                        </div>

                                        <div className="bg-base-100 p-6 rounded-lg shadow-sm mb-6">
                                            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                                                <ConfigField label="Name" description="The name of the slave (used for file generation)">
                                                    <input type="text" className="input input-bordered w-full" value={slave.name} onChange={e => updateSlave(slave.id, 'name', e.target.value)} />
                                                </ConfigField>
                                                <ConfigField label="Node ID" description="The node-ID (default: 255, can be omitted if specified in the DCF).">
                                                    <input type="number" className="input input-bordered w-full" value={slave.nodeId} onChange={e => updateSlave(slave.id, 'nodeId', parseInt(e.target.value))} />
                                                </ConfigField>
                                                <div className="col-span-1 md:col-span-2">
                                                    <ConfigField label="DCF Path" description="The filename of the EDS/DCF describing the slave (mandatory).">
                                                        <input type="text" className="input input-bordered w-full" value={slave.dcfPath} onChange={e => updateSlave(slave.id, 'dcfPath', e.target.value)} />
                                                    </ConfigField>
                                                </div>
                                                <ConfigField label="Revision Number (Hex)" description="The revision number (default: 0x00000000, can be omitted if specified in the DCF).">
                                                    <input type="text" className="input input-bordered w-full" placeholder="0x..." value={slave.revisionNumber ? slave.revisionNumber.toString(16) : ''} onChange={e => updateSlave(slave.id, 'revisionNumber', parseInt(e.target.value, 16))} />
                                                </ConfigField>
                                                <ConfigField label="Serial Number (Hex)" description="The serial number (default: 0x00000000, can be omitted if specified in the DCF).">
                                                    <input type="text" className="input input-bordered w-full" placeholder="0x..." value={slave.serialNumber ? slave.serialNumber.toString(16) : ''} onChange={e => updateSlave(slave.id, 'serialNumber', parseInt(e.target.value, 16))} />
                                                </ConfigField>
                                                <ConfigField label="Heartbeat Multiplier" description="The multiplication factor used to obtain master heartbeat consumer time from the slave heartbeat producer time (default: see options section).">
                                                    <input type="number" step="0.1" className="input input-bordered w-full" value={slave.heartbeatMultiplier} onChange={e => updateSlave(slave.id, 'heartbeatMultiplier', parseFloat(e.target.value))} />
                                                </ConfigField>
                                                <ConfigField label="Heartbeat Producer (ms)" description="The heartbeat producer time in ms (default: 0).">
                                                    <input type="number" className="input input-bordered w-full" value={slave.heartbeatProducer} onChange={e => updateSlave(slave.id, 'heartbeatProducer', parseInt(e.target.value))} />
                                                </ConfigField>
                                                <div className="col-span-1 md:col-span-2">
                                                    <ConfigField label="Software File" description="The name of the file containing the firmware (default: &quot;&quot;, see object 1F58).">
                                                        <input type="text" className="input input-bordered w-full" value={slave.softwareFile} onChange={e => updateSlave(slave.id, 'softwareFile', e.target.value)} />
                                                    </ConfigField>
                                                </div>
                                                <ConfigField label="Software Version (Hex)" description="The expected software version (default: 0x00000000, see object 1F55).">
                                                    <input type="text" className="input input-bordered w-full" placeholder="0x..." value={slave.softwareVersion ? slave.softwareVersion.toString(16) : ''} onChange={e => updateSlave(slave.id, 'softwareVersion', parseInt(e.target.value, 16))} />
                                                </ConfigField>
                                                <div className="col-span-1 md:col-span-2">
                                                    <ConfigField label="Configuration File" description="The name of the file containing the configuration (default: &quot;<dcf_path>/<name>.bin&quot;, see object 1F22).">
                                                        <input type="text" className="input input-bordered w-full" value={slave.configurationFile} onChange={e => updateSlave(slave.id, 'configurationFile', e.target.value)} />
                                                    </ConfigField>
                                                </div>
                                                <ConfigField label="Restore Configuration (Hex)" description="The sub-index of object 1011 to be used when restoring the configuration (default: 0x00).">
                                                    <input type="text" className="input input-bordered w-full" placeholder="0x..." value={slave.restoreConfiguration ? slave.restoreConfiguration.toString(16) : ''} onChange={e => updateSlave(slave.id, 'restoreConfiguration', parseInt(e.target.value, 16))} />
                                                </ConfigField>

                                                <div className="col-span-1 md:col-span-2 divider font-bold text-secondary mt-4">Flags</div>

                                                <ConfigField label="Heartbeat Consumer" description="Specifies whether the slave should monitor the heartbeat of the master (default: false).">
                                                    <input type="checkbox" className="checkbox checkbox-primary" checked={slave.heartbeatConsumer} onChange={e => updateSlave(slave.id, 'heartbeatConsumer', e.target.checked)} />
                                                </ConfigField>
                                                <ConfigField label="Boot" description="Specifies whether the slave will be configured and booted by the master (default: true, see bit 2 in object 1F81).">
                                                    <input type="checkbox" className="checkbox checkbox-primary" checked={slave.boot} onChange={e => updateSlave(slave.id, 'boot', e.target.checked)} />
                                                </ConfigField>
                                                <ConfigField label="Mandatory" description="Specifies whether the slave is mandatory (default: false, see bit 3 in object 1F81).">
                                                    <input type="checkbox" className="checkbox checkbox-primary" checked={slave.mandatory} onChange={e => updateSlave(slave.id, 'mandatory', e.target.checked)} />
                                                </ConfigField>
                                                <ConfigField label="Reset Communication" description="Specifies whether the NMT reset communication command may be sent to the slave (default: true, see bit 4 in object 1F81).">
                                                    <input type="checkbox" className="checkbox checkbox-primary" checked={slave.resetCommunication} onChange={e => updateSlave(slave.id, 'resetCommunication', e.target.checked)} />
                                                </ConfigField>
                                            </div>
                                        </div>

                                        <div className="divider font-bold text-secondary">PDOs</div>

                                        {/* RPDOs */}
                                        <div className="collapse collapse-arrow bg-base-100 shadow-sm mb-4">
                                            <input type="checkbox" />
                                            <div className="collapse-title text-lg font-medium flex justify-between items-center pr-12">
                                                <span>RPDOs ({slave.rpdos.length})</span>
                                            </div>
                                            <div className="collapse-content">
                                                <button className="btn btn-sm btn-outline btn-primary mb-4" onClick={() => addPdo(slave.id, 'RPDO')}>+ Add RPDO</button>
                                                {slave.rpdos.map((pdo) => (
                                                    <div key={pdo.id} className="card bg-base-200 mb-4 p-4 border border-base-300">
                                                        <div className="flex justify-between items-center mb-2">
                                                            <span className="font-bold text-accent">RPDO {pdo.pdoNumber}</span>
                                                            <button className="btn btn-xs btn-circle btn-ghost text-error" onClick={() => removePdo(slave.id, 'RPDO', pdo.id)}>✕</button>
                                                        </div>
                                                        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
                                                            <ConfigField label="COB-ID Type" description="The COB-ID. If the value is auto, an unused 11-bit COB-ID will be assigned.">
                                                                <select className="select select-bordered select-sm w-full" value={pdo.cobIdType} onChange={e => updatePdo(slave.id, 'RPDO', pdo.id, 'cobIdType', e.target.value)}>
                                                                    <option value="auto">Auto</option>
                                                                    <option value="manual">Manual</option>
                                                                </select>
                                                            </ConfigField>
                                                            {pdo.cobIdType === 'manual' && (
                                                                <ConfigField label="COB-ID (Hex)" description="The COB-ID. If the most significant bit is set, the PDO will be ignored by the master.">
                                                                    <input type="text" className="input input-bordered input-sm w-full" placeholder="0x..." onChange={e => updatePdo(slave.id, 'RPDO', pdo.id, 'cobId', parseInt(e.target.value, 16))} />
                                                                </ConfigField>
                                                            )}
                                                            <ConfigField label="Transmission" description="The transmission type.">
                                                                <input type="number" className="input input-bordered input-sm w-full" value={pdo.transmission} onChange={e => updatePdo(slave.id, 'RPDO', pdo.id, 'transmission', parseInt(e.target.value))} />
                                                            </ConfigField>
                                                        </div>

                                                        <div className="bg-base-100 p-3 rounded">
                                                            <div className="text-xs font-bold mb-2 uppercase tracking-wide opacity-70">Mapping</div>
                                                            {pdo.mapping.map((map, mIdx) => (
                                                                <div key={mIdx} className="grid grid-cols-[1fr_1fr_40px] gap-2 mb-2 items-end">
                                                                    <div className="form-control">
                                                                        <label className="label p-0 mb-1"><span className="label-text text-xs">Index</span></label>
                                                                        <input type="text" className="input input-bordered input-xs w-full" placeholder="Index" value={map.index.toString(16)} onChange={e => updateMapping(slave.id, 'RPDO', pdo.id, mIdx, 'index', parseInt(e.target.value, 16))} />
                                                                    </div>
                                                                    <div className="form-control">
                                                                        <label className="label p-0 mb-1"><span className="label-text text-xs">Sub</span></label>
                                                                        <input type="text" className="input input-bordered input-xs w-full" placeholder="Sub" value={map.sub_index.toString(16)} onChange={e => updateMapping(slave.id, 'RPDO', pdo.id, mIdx, 'sub_index', parseInt(e.target.value, 16))} />
                                                                    </div>
                                                                    <button className="btn btn-xs btn-ghost text-error" onClick={() => removeMapping(slave.id, 'RPDO', pdo.id, mIdx)}>✕</button>
                                                                </div>
                                                            ))}
                                                            <button className="btn btn-xs btn-ghost w-full border-dashed border border-base-300" onClick={() => addMapping(slave.id, 'RPDO', pdo.id)}>+ Add Mapping</button>
                                                        </div>
                                                    </div>
                                                ))}
                                            </div>
                                        </div>

                                        {/* TPDOs */}
                                        <div className="collapse collapse-arrow bg-base-100 shadow-sm mb-4">
                                            <input type="checkbox" />
                                            <div className="collapse-title text-lg font-medium flex justify-between items-center pr-12">
                                                <span>TPDOs ({slave.tpdos.length})</span>
                                            </div>
                                            <div className="collapse-content">
                                                <button className="btn btn-sm btn-outline btn-primary mb-4" onClick={() => addPdo(slave.id, 'TPDO')}>+ Add TPDO</button>
                                                {slave.tpdos.map((pdo) => (
                                                    <div key={pdo.id} className="card bg-base-200 mb-4 p-4 border border-base-300">
                                                        <div className="flex justify-between items-center mb-2">
                                                            <span className="font-bold text-accent">TPDO {pdo.pdoNumber}</span>
                                                            <button className="btn btn-xs btn-circle btn-ghost text-error" onClick={() => removePdo(slave.id, 'TPDO', pdo.id)}>✕</button>
                                                        </div>
                                                        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
                                                            <ConfigField label="COB-ID Type" description="The COB-ID. If the value is auto, an unused 11-bit COB-ID will be assigned.">
                                                                <select className="select select-bordered select-sm w-full" value={pdo.cobIdType} onChange={e => updatePdo(slave.id, 'TPDO', pdo.id, 'cobIdType', e.target.value)}>
                                                                    <option value="auto">Auto</option>
                                                                    <option value="manual">Manual</option>
                                                                </select>
                                                            </ConfigField>
                                                            <ConfigField label="Event Timer" description="The event timer of the (corresponding master) TPDO in ms.">
                                                                <input type="number" className="input input-bordered input-sm w-full" value={pdo.eventTimer} onChange={e => updatePdo(slave.id, 'TPDO', pdo.id, 'eventTimer', parseInt(e.target.value))} />
                                                            </ConfigField>
                                                            <ConfigField label="Sync Start" description="The SYNC start value of the (corresponding master) TPDO.">
                                                                <input type="number" className="input input-bordered input-sm w-full" value={pdo.syncStart} onChange={e => updatePdo(slave.id, 'TPDO', pdo.id, 'syncStart', parseInt(e.target.value))} />
                                                            </ConfigField>
                                                        </div>

                                                        <div className="bg-base-100 p-3 rounded">
                                                            <div className="text-xs font-bold mb-2 uppercase tracking-wide opacity-70">Mapping</div>
                                                            {pdo.mapping.map((map, mIdx) => (
                                                                <div key={mIdx} className="grid grid-cols-[1fr_1fr_40px] gap-2 mb-2 items-end">
                                                                    <div className="form-control">
                                                                        <label className="label p-0 mb-1"><span className="label-text text-xs">Index</span></label>
                                                                        <input type="text" className="input input-bordered input-xs w-full" placeholder="Index" value={map.index.toString(16)} onChange={e => updateMapping(slave.id, 'TPDO', pdo.id, mIdx, 'index', parseInt(e.target.value, 16))} />
                                                                    </div>
                                                                    <div className="form-control">
                                                                        <label className="label p-0 mb-1"><span className="label-text text-xs">Sub</span></label>
                                                                        <input type="text" className="input input-bordered input-xs w-full" placeholder="Sub" value={map.sub_index.toString(16)} onChange={e => updateMapping(slave.id, 'TPDO', pdo.id, mIdx, 'sub_index', parseInt(e.target.value, 16))} />
                                                                    </div>
                                                                    <button className="btn btn-xs btn-ghost text-error" onClick={() => removeMapping(slave.id, 'TPDO', pdo.id, mIdx)}>✕</button>
                                                                </div>
                                                            ))}
                                                            <button className="btn btn-xs btn-ghost w-full border-dashed border border-base-300" onClick={() => addMapping(slave.id, 'TPDO', pdo.id)}>+ Add Mapping</button>
                                                        </div>
                                                    </div>
                                                ))}
                                            </div>
                                        </div>

                                        {/* SDOs */}
                                        <div className="collapse collapse-arrow bg-base-100 shadow-sm mb-4">
                                            <input type="checkbox" />
                                            <div className="collapse-title text-lg font-medium flex justify-between items-center pr-12">
                                                <span>SDOs ({slave.sdos.length})</span>
                                            </div>
                                            <div className="collapse-content">
                                                <button className="btn btn-sm btn-outline btn-primary mb-4" onClick={() => addSdo(slave.id)}>+ Add SDO</button>
                                                {slave.sdos.map((sdo) => (
                                                    <div key={sdo.id} className="grid grid-cols-[100px_80px_1fr_40px] gap-4 mb-2 items-end bg-base-200 p-3 rounded border border-base-300">
                                                        <div className="form-control">
                                                            <label className="label p-0 mb-1"><span className="label-text text-xs">Index</span></label>
                                                            <input type="text" className="input input-bordered input-sm w-full" value={sdo.index.toString(16)} onChange={e => updateSdo(slave.id, sdo.id, 'index', parseInt(e.target.value, 16))} />
                                                        </div>
                                                        <div className="form-control">
                                                            <label className="label p-0 mb-1"><span className="label-text text-xs">Sub</span></label>
                                                            <input type="text" className="input input-bordered input-sm w-full" value={sdo.sub_index.toString(16)} onChange={e => updateSdo(slave.id, sdo.id, 'sub_index', parseInt(e.target.value, 16))} />
                                                        </div>
                                                        <div className="form-control">
                                                            <label className="label p-0 mb-1"><span className="label-text text-xs">Value</span></label>
                                                            <input type="number" className="input input-bordered input-sm w-full" value={sdo.value} onChange={e => updateSdo(slave.id, sdo.id, 'value', parseInt(e.target.value))} />
                                                        </div>
                                                        <button className="btn btn-sm btn-square btn-ghost text-error" onClick={() => removeSdo(slave.id, sdo.id)}>✕</button>
                                                    </div>
                                                ))}
                                            </div>
                                        </div>

                                    </div>
                                );
                            })()
                        )}
                    </div>
                </div>
            </div>

            {/* Preview Modal */}
            {showPreview && (
                <div className="modal modal-open">
                    <div className="modal-box w-11/12 max-w-5xl h-5/6 flex flex-col">
                        <h3 className="font-bold text-lg mb-4">YAML Preview</h3>
                        <pre className="bg-base-300 p-4 rounded-box flex-1 overflow-auto font-mono text-sm leading-relaxed">
                            {generateYaml()}
                        </pre>
                        <div className="modal-action">
                            <button className="btn" onClick={copyToClipboard}>Copy to Clipboard</button>
                            <button className="btn btn-primary" onClick={() => setShowPreview(false)}>Close</button>
                        </div>
                    </div>
                    <div className="modal-backdrop" onClick={() => setShowPreview(false)}></div>
                </div>
            )}
        </div>
    );
};

export default YamlGenerator;
