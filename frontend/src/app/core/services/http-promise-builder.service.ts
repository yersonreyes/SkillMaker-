import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpErrorResponse, HttpParams } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { UiDialogService } from './ui-dialog.service';

type Method = 'get' | 'post' | 'put' | 'patch' | 'delete';

class HttpRequestBuilder<T> {
  private _method: Method = 'get';
  private _url = '';
  private _body: unknown = undefined;
  private _params = new HttpParams();
  private _silent = false;

  constructor(
    private http: HttpClient,
    private dialog: UiDialogService,
  ) {}

  get(): this {
    this._method = 'get';
    return this;
  }
  post(): this {
    this._method = 'post';
    return this;
  }
  put(): this {
    this._method = 'put';
    return this;
  }
  patch(): this {
    this._method = 'patch';
    return this;
  }
  delete(): this {
    this._method = 'delete';
    return this;
  }

  url(u: string): this {
    this._url = u;
    return this;
  }

  body(b: unknown): this {
    this._body = b;
    return this;
  }

  queryParam(key: string, value: string | number | boolean | undefined | null): this {
    if (value === undefined || value === null || value === '') return this;
    this._params = this._params.set(key, String(value));
    return this;
  }

  /**
   * Appends a repeated query param for each value in the array.
   * Uses HttpParams.append (NOT .set) so multiple calls produce repeated params.
   * e.g. queryParamArray('categoria', [A, B]) → ?categoria=A&categoria=B
   * Empty or falsy values within the array are skipped.
   * An empty array emits nothing (param is omitted entirely).
   */
  queryParamArray(key: string, values: string[]): this {
    for (const v of values ?? []) {
      if (v) this._params = this._params.append(key, v);
    }
    return this;
  }

  silent(): this {
    this._silent = true;
    return this;
  }

  async send(): Promise<T> {
    try {
      const options = { params: this._params };
      let obs;
      switch (this._method) {
        case 'get':
          obs = this.http.get<T>(this._url, options);
          break;
        case 'post':
          obs = this.http.post<T>(this._url, this._body, options);
          break;
        case 'put':
          obs = this.http.put<T>(this._url, this._body, options);
          break;
        case 'patch':
          obs = this.http.patch<T>(this._url, this._body, options);
          break;
        case 'delete':
          obs = this.http.delete<T>(this._url, options);
          break;
      }
      return await firstValueFrom(obs);
    } catch (err) {
      if (!this._silent && err instanceof HttpErrorResponse) {
        const msg = (err.error as { message?: string })?.message ?? err.message;
        this.dialog.showError('Error', msg);
      }
      throw err;
    }
  }
}

@Injectable({ providedIn: 'root' })
export class HttpPromiseBuilderService {
  private http = inject(HttpClient);
  private dialog = inject(UiDialogService);

  request<T = unknown>() {
    return new HttpRequestBuilder<T>(this.http, this.dialog);
  }
}
