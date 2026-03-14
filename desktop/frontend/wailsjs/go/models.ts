export namespace adb {
	
	export class App {
	    package: string;
	    label: string;
	    icon: string;
	    hidden: boolean;
	
	    static createFrom(source: any = {}) {
	        return new App(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.package = source["package"];
	        this.label = source["label"];
	        this.icon = source["icon"];
	        this.hidden = source["hidden"];
	    }
	}

}

