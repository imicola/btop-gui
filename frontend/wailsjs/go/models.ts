export namespace main {
	
	export class ForkResult {
	    pid: number;
	    parentPid: number;
	    durationSeconds: number;
	    implementation: string;
	
	    static createFrom(source: any = {}) {
	        return new ForkResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pid = source["pid"];
	        this.parentPid = source["parentPid"];
	        this.durationSeconds = source["durationSeconds"];
	        this.implementation = source["implementation"];
	    }
	}
	export class SnapshotData {
	    timestamp: number;
	    cpu: procfs.CPUUsage;
	    memory: procfs.MemoryInfo;
	    loadAvg: number[];
	    uptime: number;
	    procs: procfs.ProcessInfo[];
	    procCount: number;
	    system: procfs.SystemInfo;
	    network: procfs.NetworkInfo;
	    disk: procfs.DiskInfo;
	
	    static createFrom(source: any = {}) {
	        return new SnapshotData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = source["timestamp"];
	        this.cpu = this.convertValues(source["cpu"], procfs.CPUUsage);
	        this.memory = this.convertValues(source["memory"], procfs.MemoryInfo);
	        this.loadAvg = source["loadAvg"];
	        this.uptime = source["uptime"];
	        this.procs = this.convertValues(source["procs"], procfs.ProcessInfo);
	        this.procCount = source["procCount"];
	        this.system = this.convertValues(source["system"], procfs.SystemInfo);
	        this.network = this.convertValues(source["network"], procfs.NetworkInfo);
	        this.disk = this.convertValues(source["disk"], procfs.DiskInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace procfs {
	
	export class CPUUsage {
	    total: number;
	    perCore: number[];
	    coreCount: number;
	
	    static createFrom(source: any = {}) {
	        return new CPUUsage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.perCore = source["perCore"];
	        this.coreCount = source["coreCount"];
	    }
	}
	export class DiskDeviceInfo {
	    name: string;
	    readBytes: number;
	    writeBytes: number;
	    readRate: number;
	    writeRate: number;
	    busyPercent: number;
	
	    static createFrom(source: any = {}) {
	        return new DiskDeviceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.readBytes = source["readBytes"];
	        this.writeBytes = source["writeBytes"];
	        this.readRate = source["readRate"];
	        this.writeRate = source["writeRate"];
	        this.busyPercent = source["busyPercent"];
	    }
	}
	export class FilesystemInfo {
	    mountPoint: string;
	    total: number;
	    used: number;
	    available: number;
	    usage: number;
	
	    static createFrom(source: any = {}) {
	        return new FilesystemInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mountPoint = source["mountPoint"];
	        this.total = source["total"];
	        this.used = source["used"];
	        this.available = source["available"];
	        this.usage = source["usage"];
	    }
	}
	export class DiskInfo {
	    available: boolean;
	    error: string;
	    root: FilesystemInfo;
	    devices: DiskDeviceInfo[];
	
	    static createFrom(source: any = {}) {
	        return new DiskInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.error = source["error"];
	        this.root = this.convertValues(source["root"], FilesystemInfo);
	        this.devices = this.convertValues(source["devices"], DiskDeviceInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class MemoryInfo {
	    total: number;
	    free: number;
	    available: number;
	    cached: number;
	    buffers: number;
	    used: number;
	    usage: number;
	    swapTotal: number;
	    swapFree: number;
	    swapUsage: number;
	
	    static createFrom(source: any = {}) {
	        return new MemoryInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.free = source["free"];
	        this.available = source["available"];
	        this.cached = source["cached"];
	        this.buffers = source["buffers"];
	        this.used = source["used"];
	        this.usage = source["usage"];
	        this.swapTotal = source["swapTotal"];
	        this.swapFree = source["swapFree"];
	        this.swapUsage = source["swapUsage"];
	    }
	}
	export class NetworkInterfaceInfo {
	    name: string;
	    rxBytes: number;
	    txBytes: number;
	    rxRate: number;
	    txRate: number;
	
	    static createFrom(source: any = {}) {
	        return new NetworkInterfaceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.rxBytes = source["rxBytes"];
	        this.txBytes = source["txBytes"];
	        this.rxRate = source["rxRate"];
	        this.txRate = source["txRate"];
	    }
	}
	export class NetworkInfo {
	    available: boolean;
	    error: string;
	    primary: string;
	    interfaces: NetworkInterfaceInfo[];
	
	    static createFrom(source: any = {}) {
	        return new NetworkInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.error = source["error"];
	        this.primary = source["primary"];
	        this.interfaces = this.convertValues(source["interfaces"], NetworkInterfaceInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ProcessDetail {
	    pid: number;
	    startTime: number;
	    user: string;
	    uid: number;
	    command: string;
	    executable: string;
	    cwd: string;
	    elapsed: number;
	    vmPeak: number;
	    vmSwap: number;
	    readBytes: number;
	    writeBytes: number;
	    voluntarySwitches: number;
	    involuntarySwitches: number;
	    fdCount: number;
	    unavailable: string[];
	
	    static createFrom(source: any = {}) {
	        return new ProcessDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pid = source["pid"];
	        this.startTime = source["startTime"];
	        this.user = source["user"];
	        this.uid = source["uid"];
	        this.command = source["command"];
	        this.executable = source["executable"];
	        this.cwd = source["cwd"];
	        this.elapsed = source["elapsed"];
	        this.vmPeak = source["vmPeak"];
	        this.vmSwap = source["vmSwap"];
	        this.readBytes = source["readBytes"];
	        this.writeBytes = source["writeBytes"];
	        this.voluntarySwitches = source["voluntarySwitches"];
	        this.involuntarySwitches = source["involuntarySwitches"];
	        this.fdCount = source["fdCount"];
	        this.unavailable = source["unavailable"];
	    }
	}
	export class ProcessInfo {
	    pid: number;
	    name: string;
	    state: string;
	    ppid: number;
	    nice: number;
	    cpu: number;
	    rss: number;
	    vsize: number;
	    threads: number;
	    startTime: number;
	    uid: number;
	    user: string;
	
	    static createFrom(source: any = {}) {
	        return new ProcessInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pid = source["pid"];
	        this.name = source["name"];
	        this.state = source["state"];
	        this.ppid = source["ppid"];
	        this.nice = source["nice"];
	        this.cpu = source["cpu"];
	        this.rss = source["rss"];
	        this.vsize = source["vsize"];
	        this.threads = source["threads"];
	        this.startTime = source["startTime"];
	        this.uid = source["uid"];
	        this.user = source["user"];
	    }
	}
	export class SystemInfo {
	    hostname: string;
	    kernel: string;
	    cpuModel: string;
	    cpuFreqMHz: number;
	
	    static createFrom(source: any = {}) {
	        return new SystemInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hostname = source["hostname"];
	        this.kernel = source["kernel"];
	        this.cpuModel = source["cpuModel"];
	        this.cpuFreqMHz = source["cpuFreqMHz"];
	    }
	}

}

