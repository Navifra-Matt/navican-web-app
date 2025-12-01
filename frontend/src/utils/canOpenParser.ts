export interface CanOpenInfo {
    type: 'NMT' | 'SYNC' | 'EMCY' | 'TPDO' | 'RPDO' | 'SDO(Req)' | 'SDO(Res)' | 'HEARTBEAT' | 'UNKNOWN';
    nodeId?: number;
    details: string;
}

export const parseCanOpenMessage = (id: number, data: number[]): CanOpenInfo => {
    // NMT Module Control
    if (id === 0x000) {
        let command = 'Unknown';
        switch (data[0]) {
            case 0x01: command = 'Start Remote Node'; break;
            case 0x02: command = 'Stop Remote Node'; break;
            case 0x80: command = 'Enter Pre-Operational'; break;
            case 0x81: command = 'Reset Node'; break;
            case 0x82: command = 'Reset Communication'; break;
        }
        const targetNode = data[1] === 0 ? 'All Nodes' : `Node ${data[1]}`;
        return { type: 'NMT', details: `${command} -> ${targetNode}` };
    }

    // SYNC
    if (id === 0x080) {
        return { type: 'SYNC', details: 'Sync Message' };
    }

    // EMCY (0x081 - 0x0FF)
    if (id >= 0x081 && id <= 0x0FF) {
        const nodeId = id - 0x080;
        const errCode = data[1] * 256 + data[0];
        const errReg = data[2];
        return { type: 'EMCY', nodeId, details: `Node ${nodeId} Error: 0x${errCode.toString(16).toUpperCase()} Reg: 0x${errReg.toString(16).toUpperCase()}` };
    }

    // PDOs
    // TPDO1 (0x181 - 0x1FF)
    if (id >= 0x181 && id <= 0x1FF) return { type: 'TPDO', nodeId: id - 0x180, details: `TPDO1 (Node ${id - 0x180})` };
    // RPDO1 (0x201 - 0x27F)
    if (id >= 0x201 && id <= 0x27F) return { type: 'RPDO', nodeId: id - 0x200, details: `RPDO1 (Node ${id - 0x200})` };
    // TPDO2 (0x281 - 0x2FF)
    if (id >= 0x281 && id <= 0x2FF) return { type: 'TPDO', nodeId: id - 0x280, details: `TPDO2 (Node ${id - 0x280})` };
    // RPDO2 (0x301 - 0x37F)
    if (id >= 0x301 && id <= 0x37F) return { type: 'RPDO', nodeId: id - 0x300, details: `RPDO2 (Node ${id - 0x300})` };
    // TPDO3 (0x381 - 0x3FF)
    if (id >= 0x381 && id <= 0x3FF) return { type: 'TPDO', nodeId: id - 0x380, details: `TPDO3 (Node ${id - 0x380})` };
    // RPDO3 (0x401 - 0x47F)
    if (id >= 0x401 && id <= 0x47F) return { type: 'RPDO', nodeId: id - 0x400, details: `RPDO3 (Node ${id - 0x400})` };
    // TPDO4 (0x481 - 0x4FF)
    if (id >= 0x481 && id <= 0x4FF) return { type: 'TPDO', nodeId: id - 0x480, details: `TPDO4 (Node ${id - 0x480})` };
    // RPDO4 (0x501 - 0x57F)
    if (id >= 0x501 && id <= 0x57F) return { type: 'RPDO', nodeId: id - 0x500, details: `RPDO4 (Node ${id - 0x500})` };

    // SDOs
    // SDO Transmit (Server -> Client) 0x581 - 0x5FF
    if (id >= 0x581 && id <= 0x5FF) {
        const nodeId = id - 0x580;
        return parseSDO(data, nodeId, 'Tx (Server->Client)');
    }
    // SDO Receive (Client -> Server) 0x601 - 0x67F
    if (id >= 0x601 && id <= 0x67F) {
        const nodeId = id - 0x600;
        return parseSDO(data, nodeId, 'Rx (Client->Server)');
    }

    // Heartbeat / Bootup (0x701 - 0x77F)
    if (id >= 0x701 && id <= 0x77F) {
        const nodeId = id - 0x700;
        const state = data[0];
        let stateStr = 'Unknown';
        switch (state) {
            case 0x00: stateStr = 'Bootup'; break;
            case 0x04: stateStr = 'Stopped'; break;
            case 0x05: stateStr = 'Operational'; break;
            case 0x7F: stateStr = 'Pre-Operational'; break;
        }
        return { type: 'HEARTBEAT', nodeId, details: `Node ${nodeId} State: ${stateStr}` };
    }

    return { type: 'UNKNOWN', details: '-' };
};

const parseSDO = (data: number[], nodeId: number, direction: string): CanOpenInfo => {
    // Determine type based on direction
    // Rx (Client->Server) is usually a Request from the Master (Client) to the Node (Server)
    // Tx (Server->Client) is usually a Response from the Node (Server) to the Master (Client)
    // Note: This naming convention depends on who is "Client" and "Server". 
    // In CANopen, the device holding the OD is the Server. The device accessing it is the Client.
    // COB-ID 0x600+NodeID is Client -> Server (Request)
    // COB-ID 0x580+NodeID is Server -> Client (Response)

    const type = direction.includes('Client->Server') ? 'SDO(Req)' : 'SDO(Res)';

    if (data.length < 8) return { type, nodeId, details: `${direction} - Invalid Length` };

    const cs = data[0];
    const index = data[1] + (data[2] << 8);
    const subIndex = data[3];
    const value = data[4] + (data[5] << 8) + (data[6] << 16) + (data[7] << 24); // Simple 32-bit LE

    let cmdStr = '';
    let valueStr = '';

    // Client -> Server (Request)
    if (type === 'SDO(Req)') {
        if (cs === 0x40) {
            cmdStr = 'Read';
            valueStr = ''; // No value in read request usually
        } else if ((cs & 0xE0) === 0x20) {
            cmdStr = 'Write';
            valueStr = `Val: 0x${value.toString(16).toUpperCase()}`;
        } else {
            cmdStr = `Req (CS: 0x${cs.toString(16)})`;
        }
    }
    // Server -> Client (Response)
    else {
        if ((cs & 0xE0) === 0x40) {
            cmdStr = 'Read Data';
            valueStr = `Val: 0x${value.toString(16).toUpperCase()}`;
        } else if (cs === 0x60) {
            cmdStr = 'Write OK';
            valueStr = 'Success';
        } else if (cs === 0x80) {
            cmdStr = 'Abort';
            valueStr = `Code: 0x${value.toString(16).toUpperCase()}`;
        } else {
            cmdStr = `Res (CS: 0x${cs.toString(16)})`;
        }
    }

    return {
        type,
        nodeId,
        details: `${cmdStr} Idx: 0x${index.toString(16).toUpperCase().padStart(4, '0')} Sub: 0x${subIndex.toString(16).toUpperCase()} ${valueStr}`
    };
};

export interface CanOpenMessageParams {
    type: 'RAW' | 'NMT' | 'SYNC' | 'TPDO' | 'RPDO' | 'SDO' | 'HEARTBEAT';
    // Common
    nodeId?: number;
    // RAW
    id?: number;
    data?: number[];
    // NMT
    nmtCommand?: number; // 0x01, 0x02, 0x80, 0x81, 0x82
    // PDO
    pdoNum?: number; // 1-4
    // SDO
    sdoType?: 'READ' | 'WRITE';
    index?: number;
    subIndex?: number;
    value?: number;
    dataLen?: number; // 1, 2, 4 bytes for Write
    // Heartbeat
    hbState?: number;
}

export const generateCanOpenMessage = (params: CanOpenMessageParams): { id: number, data: number[] } => {
    const { type, nodeId = 0 } = params;

    switch (type) {
        case 'RAW':
            return { id: params.id || 0, data: params.data || [] };

        case 'NMT':
            return {
                id: 0x000,
                data: [params.nmtCommand || 0, nodeId]
            };

        case 'SYNC':
            return { id: 0x080, data: [] };

        case 'TPDO':
        case 'RPDO': {
            const pdoNum = params.pdoNum || 1;

            let funcCode = 0;
            if (type === 'TPDO') {
                if (pdoNum === 1) funcCode = 0x180;
                else if (pdoNum === 2) funcCode = 0x280;
                else if (pdoNum === 3) funcCode = 0x380;
                else if (pdoNum === 4) funcCode = 0x480;
            } else {
                if (pdoNum === 1) funcCode = 0x200;
                else if (pdoNum === 2) funcCode = 0x300;
                else if (pdoNum === 3) funcCode = 0x400;
                else if (pdoNum === 4) funcCode = 0x500;
            }

            return {
                id: funcCode + nodeId,
                data: params.data || []
            };
        }

        case 'SDO': {
            // Client -> Server (Request) is 0x600 + NodeID
            const id = 0x600 + nodeId;
            const index = params.index || 0;
            const sub = params.subIndex || 0;
            const val = params.value || 0;
            const data = new Array(8).fill(0);

            // CS
            if (params.sdoType === 'READ') {
                data[0] = 0x40; // Upload Request
            } else {
                // Download Request
                // 1 byte: 2F, 2 bytes: 2B, 4 bytes: 23
                const len = params.dataLen || 4;
                if (len === 1) data[0] = 0x2F;
                else if (len === 2) data[0] = 0x2B;
                else data[0] = 0x23;

                // Value
                data[4] = val & 0xFF;
                data[5] = (val >> 8) & 0xFF;
                data[6] = (val >> 16) & 0xFF;
                data[7] = (val >> 24) & 0xFF;
            }

            // Index
            data[1] = index & 0xFF;
            data[2] = (index >> 8) & 0xFF;
            // Subindex
            data[3] = sub & 0xFF;

            return { id, data };
        }

        case 'HEARTBEAT':
            return {
                id: 0x700 + nodeId,
                data: [params.hbState || 0]
            };

        default:
            return { id: 0, data: [] };
    }
};
