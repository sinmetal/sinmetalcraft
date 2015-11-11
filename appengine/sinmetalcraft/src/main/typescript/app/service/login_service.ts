namespace SinmetalCraft {
    "use strict";

    import ILogin = SinmetalCraft.Model.ILogin;

    export class LoginService {
        PATH = "/api/v1/login";

        constructor(public $http:ng.IHttpService) {
        }

        get(destinationURL:string):ng.IHttpPromise<ILogin> {
            return this.$http.get(this.PATH + "?destinationURL=" + encodeURI(destinationURL));
        }
    }
}
