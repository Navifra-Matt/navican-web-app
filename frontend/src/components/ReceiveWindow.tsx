import React, { useMemo } from 'react';
import type { CanMessage } from '../hooks/useCanSocket';
import { parseCanOpenMessage } from '../utils/canOpenParser';

interface ReceiveWindowProps {
    messages: CanMessage[];
}

interface AggregatedMessage extends CanMessage {
    count: number;
    period: number;
    lastTimestamp: number;
    canOpenType: string;
    canOpenDetails: string;
    canOpenNodeId?: number;
}

const ReceiveWindow: React.FC<ReceiveWindowProps> = ({ messages }) => {
    const aggregatedMessages = useMemo(() => {
        const map = new Map<number, AggregatedMessage>();

        // Process messages from oldest to newest to calculate period correctly
        // Assuming 'messages' is ordered newest first, we reverse it for processing
        const sortedMessages = [...messages].reverse();

        sortedMessages.forEach((msg) => {
            const existing = map.get(msg.id);
            const info = parseCanOpenMessage(msg.id, msg.data);

            if (existing) {
                existing.data = msg.data;
                existing.period = msg.timestamp - existing.lastTimestamp;
                existing.lastTimestamp = msg.timestamp;
                existing.count += 1;
                existing.timestamp = msg.timestamp; // Update to latest
                existing.canOpenType = info.type;
                existing.canOpenDetails = info.details;
                existing.canOpenNodeId = info.nodeId;
            } else {
                map.set(msg.id, {
                    ...msg,
                    count: 1,
                    period: 0,
                    lastTimestamp: msg.timestamp,
                    canOpenType: info.type,
                    canOpenDetails: info.details,
                    canOpenNodeId: info.nodeId,
                });
            }
        });

        return Array.from(map.values()).sort((a, b) => a.id - b.id);
    }, [messages]);

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
                <span>Receive </span>
                <span className="text-xs font-normal opacity-70">{aggregatedMessages.length} IDs</span>
            </div>
            <div className="overflow-auto flex-1">
                <table className="table table-xs table-pin-rows font-mono w-full">
                    <thead>
                        <tr className="bg-base-200">
                            <th className="w-24">Time</th>
                            <th className="w-20">ID (Hex)</th>
                            <th className="w-16 text-center">Len</th>
                            <th>Data (Hex)</th>
                            <th className="w-16 text-center">Node</th>
                            <th className="w-24">Command</th>
                            <th>Info</th>
                            <th className="w-20 text-right">Count</th>
                            <th className="w-24 text-right">Cycle (ms)</th>
                        </tr>
                    </thead>
                    <tbody>
                        {aggregatedMessages.map((msg) => (
                            <tr key={msg.id} className="hover:bg-base-200">
                                <td>{new Date(msg.lastTimestamp).toLocaleTimeString([], { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit', fractionalSecondDigits: 3 })}</td>
                                <td>{msg.id.toString(16).toUpperCase().padStart(3, '0')}h</td>
                                <td className="text-center">{msg.data.length}</td>
                                <td>
                                    <div className="flex gap-1">
                                        {msg.data.map((byte, i) => (
                                            <span key={i} className="opacity-90">
                                                {byte.toString(16).toUpperCase().padStart(2, '0')}
                                            </span>
                                        ))}
                                    </div>
                                </td>
                                <td className="text-center opacity-70">
                                    {msg.canOpenNodeId !== undefined ? msg.canOpenNodeId : '-'}
                                </td>
                                <td>
                                    <div className={`badge badge-xs border-0 ${getBadgeColor(msg.canOpenType)} w-20`}>
                                        {msg.canOpenType}
                                    </div>
                                </td>
                                <td className="text-xs opacity-70 truncate max-w-xs" title={msg.canOpenDetails}>
                                    {msg.canOpenDetails}
                                </td>
                                <td className="text-right">{msg.count}</td>
                                <td className="text-right">{msg.period}</td>
                            </tr>
                        ))}
                        {aggregatedMessages.length === 0 && (
                            <tr>
                                <td colSpan={9} className="text-center py-8 opacity-50">
                                    No messages received
                                </td>
                            </tr>
                        )}
                    </tbody>
                </table>
            </div>
        </div>
    );
};

export default ReceiveWindow;
