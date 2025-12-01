import { useState, useEffect, useRef } from 'react';

export interface CanMessage {
    id: number;
    data: number[];
    timestamp: number;
}

export interface UseCanSocketReturn {
    lastMessage: CanMessage | null;
    isConnected: boolean;
    messages: CanMessage[];
    sendMessage: (id: number, data: number[]) => void;
}

const useCanSocket = (url: string): UseCanSocketReturn => {
    const [lastMessage, setLastMessage] = useState<CanMessage | null>(null);
    const [isConnected, setIsConnected] = useState(false);
    const [messages, setMessages] = useState<CanMessage[]>([]);
    const socketRef = useRef<WebSocket | null>(null);

    const sendMessage = (id: number, data: number[]) => {
        if (socketRef.current && socketRef.current.readyState === WebSocket.OPEN) {
            const message = {
                id,
                data,
                timestamp: Date.now()
            };
            socketRef.current.send(JSON.stringify(message));
        } else {
            console.warn('WebSocket is not connected. Cannot send message.');
        }
    };

    useEffect(() => {
        const connect = () => {
            const socket = new WebSocket(url);
            socketRef.current = socket;

            socket.onopen = () => {
                console.log('WebSocket connected');
                setIsConnected(true);
            };

            socket.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    // Validate data structure roughly
                    if (typeof data.id === 'number' && Array.isArray(data.data)) {
                        const newMessage: CanMessage = {
                            id: data.id,
                            data: data.data,
                            timestamp: Date.now(),
                        };
                        setLastMessage(newMessage);
                        setMessages((prev) => [newMessage, ...prev].slice(0, 1000)); // Keep last 1000 messages
                    }
                } catch (error) {
                    console.error('Error parsing WebSocket message:', error);
                }
            };

            socket.onclose = () => {
                console.log('WebSocket disconnected');
                setIsConnected(false);
                // Reconnect logic could go here
                setTimeout(connect, 3000);
            };

            socket.onerror = (error) => {
                console.error('WebSocket error:', error);
                socket.close();
            };
        };

        connect();

        // Mock data generator for demonstration
        const mockInterval = setInterval(() => {
            if (!isConnected) {
                // Generate various CANopen messages
                const msgTypes = [
                    // NMT Start Node 1
                    { id: 0x000, data: [0x01, 0x01] },
                    // SYNC
                    { id: 0x080, data: [] },
                    // EMCY Node 1 (Error 0x1000, Reg 0x01)
                    { id: 0x081, data: [0x00, 0x10, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00] },
                    // TPDO1 Node 1
                    { id: 0x181, data: [0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08] },
                    // RPDO1 Node 1
                    { id: 0x201, data: [0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10, 0x11] },
                    // RPDO2 Node 1
                    { id: 0x301, data: [0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27] },
                    // SDO Request (Read Index 0x1000 Sub 0)
                    { id: 0x601, data: [0x40, 0x00, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00] },
                    // SDO Response (Read Success, Value 0x12345678)
                    { id: 0x581, data: [0x43, 0x00, 0x10, 0x00, 0x78, 0x56, 0x34, 0x12] },
                    // Heartbeat Node 1 (Operational)
                    { id: 0x701, data: [0x05] },
                    // Random Noise
                    { id: 0x123, data: [0xAA, 0xBB] }
                ];

                const randomMsg = msgTypes[Math.floor(Math.random() * msgTypes.length)];

                const mockMsg: CanMessage = {
                    id: randomMsg.id,
                    data: randomMsg.data,
                    timestamp: Date.now(),
                };
                setLastMessage(mockMsg);
                setMessages((prev) => [mockMsg, ...prev].slice(0, 1000));
            }
        }, 500); // Slower interval for readability

        return () => {
            clearInterval(mockInterval);
            if (socketRef.current) {
                socketRef.current.close();
            }
        };
    }, [url, isConnected]);

    return { lastMessage, isConnected, messages, sendMessage };
};

export default useCanSocket;
