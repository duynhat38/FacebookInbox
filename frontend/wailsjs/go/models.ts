export namespace domain {
	
	export class AccountItem {
	    id: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new AccountItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.value = source["value"];
	    }
	}
	export class AccountSnapshot {
	    id: string;
	    value: string;
	    proxy: string;
	    success: number;
	    errors: number;
	    log: string;
	    running: boolean;
	    started: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AccountSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.value = source["value"];
	        this.proxy = source["proxy"];
	        this.success = source["success"];
	        this.errors = source["errors"];
	        this.log = source["log"];
	        this.running = source["running"];
	        this.started = source["started"];
	    }
	}
	export class Settings {
	    threads: number;
	    delaySeconds: number;
	    maxErrors: number;
	    pagesPerThread: number;
	    accountPerLink: number;
	    imagePath: string;
	    language: string;
	    deleteThreadAfterSend: boolean;
	    deleteMessageAfterSend: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.threads = source["threads"];
	        this.delaySeconds = source["delaySeconds"];
	        this.maxErrors = source["maxErrors"];
	        this.pagesPerThread = source["pagesPerThread"];
	        this.accountPerLink = source["accountPerLink"];
	        this.imagePath = source["imagePath"];
	        this.language = source["language"];
	        this.deleteThreadAfterSend = source["deleteThreadAfterSend"];
	        this.deleteMessageAfterSend = source["deleteMessageAfterSend"];
	    }
	}
	export class Stats {
	    state: string;
	    totalWorkers: number;
	    running: number;
	    success: number;
	    errors: number;
	    remaining: number;
	    totalPages: number;
	    processed: number;
	    accounts: AccountSnapshot[];
	
	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.totalWorkers = source["totalWorkers"];
	        this.running = source["running"];
	        this.success = source["success"];
	        this.errors = source["errors"];
	        this.remaining = source["remaining"];
	        this.totalPages = source["totalPages"];
	        this.processed = source["processed"];
	        this.accounts = this.convertValues(source["accounts"], AccountSnapshot);
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

