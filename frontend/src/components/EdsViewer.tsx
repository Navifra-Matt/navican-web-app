import React, { useState, useMemo } from 'react';
import { FileText, Info } from 'lucide-react';

interface EdsSection {
    name: string;
    entries: { key: string; value: string }[];
}

interface ObjectDetails {
    id: string;
    section: EdsSection | undefined;
    subSections: EdsSection[];
}

type SelectionType = 'deviceInfo' | { type: 'object', id: string };

const EdsViewer: React.FC = () => {
    const [sections, setSections] = useState<EdsSection[]>([]);
    const [deviceInfo, setDeviceInfo] = useState<EdsSection | undefined>(undefined);
    const [mandatoryObjects, setMandatoryObjects] = useState<ObjectDetails[]>([]);
    const [optionalObjects, setOptionalObjects] = useState<ObjectDetails[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [fileName, setFileName] = useState<string | null>(null);
    const [selection, setSelection] = useState<SelectionType | null>(null);
    const [searchTerm, setSearchTerm] = useState('');

    const parseEds = (content: string) => {
        try {
            const parsedSections: EdsSection[] = [];
            let currentSection: EdsSection | null = null;

            const lines = content.split('\n');

            for (const line of lines) {
                const trimmedLine = line.trim();
                if (!trimmedLine || trimmedLine.startsWith(';')) continue;

                if (trimmedLine.startsWith('[') && trimmedLine.endsWith(']')) {
                    if (currentSection) {
                        parsedSections.push(currentSection);
                    }
                    currentSection = {
                        name: trimmedLine.slice(1, -1),
                        entries: []
                    };
                } else if (currentSection && trimmedLine.includes('=')) {
                    const [key, ...valueParts] = trimmedLine.split('=');
                    currentSection.entries.push({
                        key: key.trim(),
                        value: valueParts.join('=').trim()
                    });
                }
            }
            if (currentSection) {
                parsedSections.push(currentSection);
            }

            setSections(parsedSections);

            const findSection = (name: string) => parsedSections.find(s => s.name === name);

            setDeviceInfo(findSection('DeviceInfo'));

            const extractObjectIds = (sectionName: string): string[] => {
                const section = findSection(sectionName);
                if (!section) return [];
                return section.entries
                    .filter(e => e.key !== 'SupportedObjects')
                    .map(e => {
                        const hexValue = e.value.toLowerCase();
                        if (hexValue.startsWith('0x')) {
                            return hexValue.substring(2).toUpperCase();
                        }
                        return e.value.toUpperCase();
                    });
            };

            const mapIdsToDetails = (ids: string[]): ObjectDetails[] => {
                return ids.map(id => {
                    const section = findSection(id);

                    // Find sub-indices
                    // Pattern: [IDsubX] where ID is the object ID and X is the sub-index
                    // e.g. [1400sub0], [1400sub1]
                    const subSections = parsedSections.filter(s =>
                        s.name.startsWith(`${id}sub`)
                    ).sort((a, b) => {
                        // Sort by sub-index number
                        const subA = parseInt(a.name.replace(`${id}sub`, ''), 16);
                        const subB = parseInt(b.name.replace(`${id}sub`, ''), 16);
                        return subA - subB;
                    });

                    return {
                        id,
                        section,
                        subSections
                    };
                });
            };

            setMandatoryObjects(mapIdsToDetails(extractObjectIds('MandatoryObjects')));
            setOptionalObjects(mapIdsToDetails(extractObjectIds('OptionalObjects')));

            // Default selection
            if (findSection('DeviceInfo')) {
                setSelection('deviceInfo');
            } else {
                setSelection(null);
            }

            setLoading(false);
        } catch (err) {
            console.error("Error parsing EDS file:", err);
            setError("Failed to parse EDS file.");
            setLoading(false);
        }
    };

    const handleFileUpload = (event: React.ChangeEvent<HTMLInputElement>) => {
        const file = event.target.files?.[0];
        if (!file) return;

        setLoading(true);
        setError(null);
        setFileName(file.name);

        const reader = new FileReader();
        reader.onload = (e) => {
            const content = e.target?.result as string;
            parseEds(content);
        };
        reader.onerror = () => {
            setError("Failed to read file.");
            setLoading(false);
        };
        reader.readAsText(file);
    };

    const filterObjects = (objects: ObjectDetails[]) => {
        if (!searchTerm) return objects;
        const lowerTerm = searchTerm.toLowerCase();
        return objects.filter(obj => {
            // Check Object ID
            if (obj.id.toLowerCase().includes(lowerTerm)) return true;

            // Check ParameterName
            const paramName = obj.section?.entries.find(e => e.key === 'ParameterName')?.value.toLowerCase();
            if (paramName && paramName.includes(lowerTerm)) return true;

            // Check Sub-Indices
            return obj.subSections.some(sub => {
                // Check Sub-Index ID (e.g. sub0)
                if (sub.name.toLowerCase().includes(lowerTerm)) return true;
                // Check Sub-Index ParameterName
                const subParamName = sub.entries.find(e => e.key === 'ParameterName')?.value.toLowerCase();
                if (subParamName && subParamName.includes(lowerTerm)) return true;
                return false;
            });
        });
    };

    const filteredMandatory = useMemo(() => filterObjects(mandatoryObjects), [mandatoryObjects, searchTerm]);
    const filteredOptional = useMemo(() => filterObjects(optionalObjects), [optionalObjects, searchTerm]);

    const renderDetails = () => {
        if (!selection) return <div className="text-base-content/50 italic">Select an item to view details</div>;

        let title = "";
        let data: { key: string; value: string }[] = [];
        let subSections: EdsSection[] = [];

        if (selection === 'deviceInfo') {
            title = "Device Info";
            data = deviceInfo?.entries || [];
        } else if (selection.type === 'object') {
            const obj = [...mandatoryObjects, ...optionalObjects].find(o => o.id === selection.id);
            title = `Object 0x${selection.id}`;
            if (obj) {
                data = obj.section?.entries || [];
                subSections = obj.subSections;
            } else {
                return (
                    <div>
                        <h2 className="text-2xl font-bold text-primary mb-4">{title}</h2>
                        <div className="alert alert-warning">Definition not found in EDS file.</div>
                    </div>
                );
            }
        }

        return (
            <div>
                <h2 className="text-2xl font-bold text-primary mb-6 border-b border-base-300 pb-2">{title}</h2>

                {/* Main Object Details */}
                <div className="overflow-x-auto bg-base-100 rounded-lg shadow mb-8">
                    <table className="table w-full">
                        <tbody>
                            {data.map((entry, index) => (
                                <tr key={index} className="hover">
                                    <td className="font-semibold text-base-content/70 w-1/3">{entry.key}</td>
                                    <td className="font-mono text-sm">{entry.value}</td>
                                </tr>
                            ))}
                            {data.length === 0 && (
                                <tr>
                                    <td colSpan={2} className="text-center text-base-content/50 py-4">No data available</td>
                                </tr>
                            )}
                        </tbody>
                    </table>
                </div>

                {/* Sub-Indices */}
                {subSections.length > 0 && (
                    <div>
                        <h3 className="text-xl font-bold text-secondary mb-4">Sub-Indices</h3>
                        <div className="grid gap-6">
                            {subSections.map((sub, index) => (
                                <div key={index} className="card bg-base-100 shadow-sm border border-base-200">
                                    <div className="card-body p-4">
                                        <h4 className="card-title text-sm text-accent uppercase tracking-wider mb-2">
                                            Sub-Index {sub.name.split('sub')[1]} (0x{parseInt(sub.name.split('sub')[1], 16).toString(16).toUpperCase().padStart(2, '0')})
                                        </h4>
                                        <div className="overflow-x-auto">
                                            <table className="table table-xs w-full">
                                                <tbody>
                                                    {sub.entries.map((entry, entryIndex) => (
                                                        <tr key={entryIndex} className="hover">
                                                            <td className="font-semibold text-base-content/70 w-1/3">{entry.key}</td>
                                                            <td className="font-mono text-xs">{entry.value}</td>
                                                        </tr>
                                                    ))}
                                                </tbody>
                                            </table>
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>
                )}
            </div>
        );
    };

    return (
        <div className="flex flex-col h-full bg-base-200">
            {/* Header / Toolbar */}
            <div className="bg-base-100 border-b border-base-300 p-4 flex justify-between items-center shadow-sm z-10">
                <h1 className="text-xl font-bold text-primary flex items-center gap-2">
                    <FileText className="h-6 w-6" />
                    EDS Viewer
                </h1>
                <div className="flex items-center gap-4">
                    {fileName && <span className="text-sm font-semibold opacity-70 bg-base-200 px-3 py-1 rounded-full">{fileName}</span>}
                    <input
                        type="file"
                        accept=".eds"
                        className="file-input file-input-sm file-input-bordered file-input-primary w-full max-w-xs"
                        onChange={handleFileUpload}
                    />
                </div>
            </div>

            {/* Main Content */}
            <div className="flex-1 flex overflow-hidden">
                {loading && <div className="flex-1 flex items-center justify-center">Loading...</div>}
                {error && <div className="flex-1 flex items-center justify-center text-error">{error}</div>}

                {!loading && !error && sections.length === 0 && (
                    <div className="flex-1 flex flex-col items-center justify-center text-base-content/50">
                        <svg xmlns="http://www.w3.org/2000/svg" className="h-16 w-16 mb-4 opacity-20" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
                        </svg>
                        <p className="text-lg">Upload an EDS file to begin</p>
                    </div>
                )}

                {!loading && !error && sections.length > 0 && (
                    <>
                        {/* Left Pane: Navigation Tree */}
                        <div className="w-80 bg-base-100 border-r border-base-300 flex flex-col overflow-hidden">
                            <div className="p-2 border-b border-base-200">
                                <input
                                    type="text"
                                    placeholder="Search ID, Name, Sub..."
                                    className="input input-sm input-bordered w-full"
                                    value={searchTerm}
                                    onChange={(e) => setSearchTerm(e.target.value)}
                                />
                            </div>
                            <div className="flex-1 overflow-y-auto">
                                <ul className="menu w-full p-2">
                                    <li>
                                        <button
                                            className={selection === 'deviceInfo' ? 'active' : ''}
                                            onClick={() => setSelection('deviceInfo')}
                                        >
                                            <Info className="h-5 w-5" />
                                            Device Info
                                        </button>
                                    </li>

                                    {filteredMandatory.length > 0 && (
                                        <>
                                            <li className="menu-title mt-4">
                                                <span>Mandatory Objects ({filteredMandatory.length})</span>
                                            </li>
                                            {filteredMandatory.map(obj => (
                                                <li key={obj.id}>
                                                    <button
                                                        className={selection && typeof selection === 'object' && selection.type === 'object' && selection.id === obj.id ? 'active' : ''}
                                                        onClick={() => setSelection({ type: 'object', id: obj.id })}
                                                    >
                                                        <span className="font-mono text-xs opacity-70">0x{obj.id}</span>
                                                        <span className="truncate text-xs">
                                                            {obj.section?.entries.find(e => e.key === 'ParameterName')?.value || 'Unknown'}
                                                        </span>
                                                    </button>
                                                </li>
                                            ))}
                                        </>
                                    )}

                                    {filteredOptional.length > 0 && (
                                        <>
                                            <li className="menu-title mt-4">
                                                <span>Optional Objects ({filteredOptional.length})</span>
                                            </li>
                                            {filteredOptional.map(obj => (
                                                <li key={obj.id}>
                                                    <button
                                                        className={selection && typeof selection === 'object' && selection.type === 'object' && selection.id === obj.id ? 'active' : ''}
                                                        onClick={() => setSelection({ type: 'object', id: obj.id })}
                                                    >
                                                        <span className="font-mono text-xs opacity-70">0x{obj.id}</span>
                                                        <span className="truncate text-xs">
                                                            {obj.section?.entries.find(e => e.key === 'ParameterName')?.value || 'Unknown'}
                                                        </span>
                                                    </button>
                                                </li>
                                            ))}
                                        </>
                                    )}
                                </ul>
                            </div>
                        </div>

                        {/* Right Pane: Details */}
                        <div className="flex-1 bg-base-200 p-8 overflow-y-auto">
                            <div className="max-w-4xl mx-auto">
                                {renderDetails()}
                            </div>
                        </div>
                    </>
                )}
            </div>
        </div>
    );
};

export default EdsViewer;
