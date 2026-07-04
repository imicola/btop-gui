export namespace main {
	
	export class SnapshotData {
	    timestamp: number;
	    cpu: procfs.CPUUsage;
	    memory: procfs.MemoryInfo;
	    loadAvg: number[];
	    uptime: number;
	    procs: procfs.ProcessInfo[];
	    procCount: number;
	
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
	    }
	}

}

