import React, { useState, useEffect, useRef } from 'react';
import { generateCanOpenMessage, parseCanOpenMessage } from '../utils/canOpenParser';
import type { CanOpenMessageParams } from '../utils/canOpenParser';

interface TransmitMessage {
    uuid: string;
    canId: number;
    data: number[]; // Array of bytes
    length: number;
    cycleTime: number; // ms
    isActive: boolean;
    count: number;
    description?: string;
}

interface TransmitWindowProps {
    sendMessage: (id: number, data: number[]) => void;
}

const TransmitWindow: React.FC<TransmitWindowProps> = ({ sendMessage }) => {
    const [messages, setMessages] = useState<TransmitMessage[]>([
        {
            uuid: '1',
            canId: 0x123,
            data: [0x11, 0x22, 0x33, 0x44],
            length: 4,
            cycleTime: 1000,
            isActive: false,
            count: 0,
            description: 'Example Msg'
        },
    ]);

    const [isModalOpen, setIsModalOpen] = useState(false);

    // Builder State
    const [msgType, setMsgType] = useState<CanOpenMessageParams['type']>('RAW');
    const [rawId, setRawId] = useState('100');
    const [rawData, setRawData] = useState('00 00 00 00');
    const [cycleTime, setCycleTime] = useState(1000);

    // CANopen Fields
    const [nodeId, setNodeId] = useState(1);
    const [nmtCmd, setNmtCmd] = useState(0x01);
    const [pdoNum, setPdoNum] = useState(1);
    const [sdoType, setSdoType] = useState<'READ' | 'WRITE'>('READ');
    const [index, setIndex] = useState('1000');
    const [subIndex, setSubIndex] = useState('00');
    const [sdoValue, setSdoValue] = useState('00000000');
    const [hbState, setHbState] = useState(0x05);

    // Refs for intervals
    const intervalsRef = useRef<{ [key: string]: ReturnType<typeof setInterval> }>({});

    useEffect(() => {
        // Cleanup intervals on unmount
        return () => {
            Object.values(intervalsRef.current).forEach(clearInterval);
        };
    }, []);

    const handleSend = (uuid: string) => {
        setMessages((prev) =>
            prev.map((msg) => {
                if (msg.uuid === uuid) {
                    sendMessage(msg.canId, msg.data);
                    return { ...msg, count: msg.count + 1 };
                }
                return msg;
            })
        );
    };

    const togglePeriodic = (uuid: string) => {
        setMessages((prev) => {
            const newMessages = prev.map((msg) => {
                if (msg.uuid === uuid) {
                    const newActive = !msg.isActive;

                    if (newActive) {
                        // Start interval
                        if (intervalsRef.current[uuid]) clearInterval(intervalsRef.current[uuid]);
                        intervalsRef.current[uuid] = setInterval(() => {
                            handleSend(uuid);
                        }, msg.cycleTime);
                    } else {
                        // Stop interval
                        if (intervalsRef.current[uuid]) {
                            clearInterval(intervalsRef.current[uuid]);
                            delete intervalsRef.current[uuid];
                        }
                    }

                    return { ...msg, isActive: newActive };
                }
                return msg;
            });
            return newMessages;
        });
    };

    const handleAddMessage = () => {
        let canId = 0;
        let data: number[] = [];
        let description = '';

        if (msgType === 'RAW') {
            canId = parseInt(rawId, 16);
            data = rawData.split(' ').map(b => parseInt(b, 16)).filter(n => !isNaN(n));
            description = 'Raw Message';
        } else {
            const params: CanOpenMessageParams = {
                type: msgType,
                nodeId,
                nmtCommand: nmtCmd,
                pdoNum,
                sdoType,
                index: parseInt(index, 16),
                subIndex: parseInt(subIndex, 16),
                value: parseInt(sdoValue, 16),
                hbState,
                data: rawData.split(' ').map(b => parseInt(b, 16)).filter(n => !isNaN(n)) // For PDO data
            };
            const result = generateCanOpenMessage(params);
            canId = result.id;
            data = result.data;
            description = `${msgType} Msg`;
        }

        const newMessage: TransmitMessage = {
            uuid: Date.now().toString(),
            canId,
            data,
            length: data.length,
            cycleTime,
            isActive: false,
            count: 0,
            description
        };

        setMessages([...messages, newMessage]);
        setIsModalOpen(false);
    };

    const handleDelete = (uuid: string) => {
        if (intervalsRef.current[uuid]) {
            clearInterval(intervalsRef.current[uuid]);
            delete intervalsRef.current[uuid];
        }
        setMessages(messages.filter(m => m.uuid !== uuid));
    };

    const getBadgeColor = (type: string) => {
        switch (type) {
            case 'NMT': return 'bg-indigo-900 text-white';
            case 'SYNC': return 'bg-gray-300 text-black';
            case 'EMCY': return 'bg-red-600 text-white';
            case 'TPDO': return 'bg-cyan-500 text-black';
            case 'RPDO': return 'bg-cyan-700 text-white';
            case 'SDO(Req)': return 'bg-yellow-400 text-black';
            case 'SDO(Res)': return 'bg-orange-600 text-white';
            case 'HEARTBEAT': return 'bg-slate-400 text-black';
            default: return 'bg-gray-500 text-white';
        }
    };

    return (
        <div className="flex flex-col h-full bg-base-100 rounded-box shadow overflow-hidden border border-base-300">
            <div className="bg-base-200 p-2 font-bold border-b border-base-300 flex justify-between items-center">
                <span>Transmit</span>
                <button
                    className="btn btn-xs btn-primary"
                    onClick={() => setIsModalOpen(true)}
                >
                    + New Message
                </button>
            </div>
            <div className="overflow-auto flex-1">
                <table className="table table-xs table-pin-rows font-mono w-full">
                    <thead>
                        <tr className="bg-base-200">
                            <th className="w-20">ID (Hex)</th>
                            <th className="w-16">Len</th>
                            <th>Data (Hex)</th>
                            <th className="w-16 text-center">Node</th>
                            <th className="w-24">Command</th>
                            <th>Info</th>
                            <th className="w-20 text-right">Count</th>
                            <th className="w-24 text-right">Cycle (ms)</th>
                            <th className="w-16 text-center">Send</th>
                            <th className="w-16 text-center">Active</th>
                            <th className="w-16">Action</th>
                        </tr>
                    </thead>
                    <tbody>
                        {messages.map((msg) => {
                            const info = parseCanOpenMessage(msg.canId, msg.data);
                            return (
                                <tr key={msg.uuid} className="hover:bg-base-200">
                                    <td>{msg.canId.toString(16).toUpperCase().padStart(3, '0')}h</td>
                                    <td>{msg.length}</td>
                                    <td>
                                        <div className="flex gap-1 items-center">
                                            {msg.data.map((byte, i) => (
                                                <span key={i} className="opacity-90">
                                                    {byte.toString(16).toUpperCase().padStart(2, '0')}
                                                </span>
                                            ))}
                                        </div>
                                    </td>
                                    <td className="text-center opacity-70">
                                        {info.nodeId !== undefined ? info.nodeId : '-'}
                                    </td>
                                    <td>
                                        <div className={`badge badge-xs border-0 ${getBadgeColor(info.type)} w-20`}>
                                            {info.type}
                                        </div>
                                    </td>
                                    <td className="text-xs opacity-70 truncate max-w-xs" title={info.details}>
                                        {info.details}
                                    </td>
                                    <td className="text-right">{msg.count}</td>
                                    <td className="text-right">{msg.cycleTime}</td>
                                    <td className="text-center">
                                        <button
                                            className="btn btn-xs btn-ghost btn-square"
                                            onClick={() => handleSend(msg.uuid)}
                                            title="Send Once"
                                        >
                                            ▶
                                        </button>
                                    </td>
                                    <td className="text-center">
                                        <input
                                            type="checkbox"
                                            className="checkbox checkbox-xs"
                                            checked={msg.isActive}
                                            onChange={() => togglePeriodic(msg.uuid)}
                                        />
                                    </td>
                                    <td>
                                        <button
                                            className="btn btn-xs btn-ghost text-error"
                                            onClick={() => handleDelete(msg.uuid)}
                                        >
                                            ×
                                        </button>
                                    </td>
                                </tr>
                            );
                        })}
                    </tbody>
                </table>
            </div>

            {/* Modal */}
            {isModalOpen && (
                <div className="modal modal-open">
                    <div className="modal-box">
                        <h3 className="font-bold text-lg mb-4">Add New Message</h3>

                        {/* Type Selector */}
                        <div className="form-control w-full mb-4">
                            <label className="label"><span className="label-text">Message Type</span></label>
                            <select
                                className="select select-bordered select-sm w-full"
                                value={msgType}
                                onChange={(e) => setMsgType(e.target.value as any)}
                            >
                                <option value="RAW">Raw Message</option>
                                <option value="NMT">NMT (Network Management)</option>
                                <option value="SYNC">SYNC</option>
                                <option value="TPDO">TPDO (Transmit PDO)</option>
                                <option value="RPDO">RPDO (Receive PDO)</option>
                                <option value="SDO">SDO (Service Data Object)</option>
                                <option value="HEARTBEAT">Heartbeat</option>
                            </select>
                        </div>

                        {/* Dynamic Fields */}
                        <div className="grid grid-cols-2 gap-2">
                            {/* Common Node ID for most types */}
                            {msgType !== 'RAW' && msgType !== 'SYNC' && (
                                <div className="form-control w-full">
                                    <label className="label"><span className="label-text">Node ID (Dec)</span></label>
                                    <input type="number" className="input input-bordered input-sm" value={nodeId} onChange={e => setNodeId(parseInt(e.target.value))} />
                                </div>
                            )}

                            {/* RAW Fields */}
                            {msgType === 'RAW' && (
                                <>
                                    <div className="form-control w-full">
                                        <label className="label"><span className="label-text">ID (Hex)</span></label>
                                        <input type="text" className="input input-bordered input-sm" value={rawId} onChange={e => setRawId(e.target.value)} />
                                    </div>
                                    <div className="form-control w-full col-span-2">
                                        <label className="label"><span className="label-text">Data (Hex)</span></label>
                                        <input type="text" className="input input-bordered input-sm" value={rawData} onChange={e => setRawData(e.target.value)} />
                                    </div>
                                </>
                            )}

                            {/* NMT Fields */}
                            {msgType === 'NMT' && (
                                <div className="form-control w-full">
                                    <label className="label"><span className="label-text">Command</span></label>
                                    <select className="select select-bordered select-sm" value={nmtCmd} onChange={e => setNmtCmd(parseInt(e.target.value))}>
                                        <option value={0x01}>Start Remote Node</option>
                                        <option value={0x02}>Stop Remote Node</option>
                                        <option value={0x80}>Enter Pre-Operational</option>
                                        <option value={0x81}>Reset Node</option>
                                        <option value={0x82}>Reset Communication</option>
                                    </select>
                                </div>
                            )}

                            {/* PDO Fields */}
                            {(msgType === 'TPDO' || msgType === 'RPDO') && (
                                <>
                                    <div className="form-control w-full">
                                        <label className="label"><span className="label-text">PDO Number</span></label>
                                        <select className="select select-bordered select-sm" value={pdoNum} onChange={e => setPdoNum(parseInt(e.target.value))}>
                                            <option value={1}>1</option>
                                            <option value={2}>2</option>
                                            <option value={3}>3</option>
                                            <option value={4}>4</option>
                                        </select>
                                    </div>
                                    <div className="form-control w-full col-span-2">
                                        <label className="label"><span className="label-text">Data (Hex)</span></label>
                                        <input type="text" className="input input-bordered input-sm" value={rawData} onChange={e => setRawData(e.target.value)} />
                                    </div>
                                </>
                            )}

                            {/* SDO Fields */}
                            {msgType === 'SDO' && (
                                <>
                                    <div className="form-control w-full">
                                        <label className="label"><span className="label-text">Type</span></label>
                                        <select className="select select-bordered select-sm" value={sdoType} onChange={e => setSdoType(e.target.value as any)}>
                                            <option value="READ">Read (Upload)</option>
                                            <option value="WRITE">Write (Download)</option>
                                        </select>
                                    </div>
                                    <div className="form-control w-full">
                                        <label className="label"><span className="label-text">Index (Hex)</span></label>
                                        <input type="text" className="input input-bordered input-sm" value={index} onChange={e => setIndex(e.target.value)} />
                                    </div>
                                    <div className="form-control w-full">
                                        <label className="label"><span className="label-text">SubIndex (Hex)</span></label>
                                        <input type="text" className="input input-bordered input-sm" value={subIndex} onChange={e => setSubIndex(e.target.value)} />
                                    </div>
                                    {sdoType === 'WRITE' && (
                                        <div className="form-control w-full">
                                            <label className="label"><span className="label-text">Value (Hex, LE)</span></label>
                                            <input type="text" className="input input-bordered input-sm" value={sdoValue} onChange={e => setSdoValue(e.target.value)} />
                                        </div>
                                    )}
                                </>
                            )}

                            {/* Heartbeat Fields */}
                            {msgType === 'HEARTBEAT' && (
                                <div className="form-control w-full">
                                    <label className="label"><span className="label-text">State</span></label>
                                    <select className="select select-bordered select-sm" value={hbState} onChange={e => setHbState(parseInt(e.target.value))}>
                                        <option value={0x00}>Bootup</option>
                                        <option value={0x04}>Stopped</option>
                                        <option value={0x05}>Operational</option>
                                        <option value={0x7F}>Pre-Operational</option>
                                    </select>
                                </div>
                            )}
                        </div>

                        <div className="form-control w-full max-w-xs mt-4">
                            <label className="label"><span className="label-text">Cycle Time (ms)</span></label>
                            <input
                                type="number"
                                className="input input-bordered input-sm"
                                value={cycleTime}
                                onChange={e => setCycleTime(parseInt(e.target.value))}
                            />
                        </div>

                        <div className="modal-action">
                            <button className="btn btn-sm" onClick={() => setIsModalOpen(false)}>Cancel</button>
                            <button className="btn btn-sm btn-primary" onClick={handleAddMessage}>Add</button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};

export default TransmitWindow;
