import type * as Api from 'common/api';

jest.mock('common/api', () => ({
  getUser: jest.fn(),
  approveDeleteMe: jest.fn(),
}));

const payload = '<img src=x onerror=window.__deletemeInjected=1>';

/** loads the entry script against a fresh root node and waits for it to finish rendering */
async function runPage(): Promise<HTMLElement> {
  process.env.REMARK_NODE = 'remark42';
  document.body.innerHTML = '<div id="remark42"></div>';
  const root = document.getElementById('remark42')!;

  await jest.isolateModulesAsync(async () => {
    await import('./deleteme');
  });
  // getUser and approveDeleteMe resolve on separate turns before the page renders
  for (let i = 0; i < 5; i++) {
    await Promise.resolve();
  }

  return root;
}

function mockedApi(): jest.Mocked<typeof Api> {
  return jest.requireMock('common/api');
}

describe('deleteme page', () => {
  beforeEach(() => {
    jest.resetModules();
    jest.clearAllMocks();
    jest.spyOn(console, 'error').mockImplementation(() => {});
    window.history.replaceState(null, '', '/web/deleteme.html?token=t');
    mockedApi().getUser.mockResolvedValue({ admin: true } as Awaited<ReturnType<typeof Api.getUser>>);
  });

  it('shows an error from the server as text', async () => {
    // the shape the backend returns when the token names a site it does not serve
    mockedApi().approveDeleteMe.mockRejectedValue({ code: 0, error: `site "${payload}" not found` });

    const root = await runPage();

    expect(root.querySelector('img')).toBeNull();
    expect(root.querySelector('pre')?.textContent).toBe(`site "${payload}" not found`);
  });

  it('shows the deletion result as text', async () => {
    mockedApi().approveDeleteMe.mockResolvedValue({ user_id: 'dev_user', site_id: [payload] } as never);

    const root = await runPage();

    expect(root.querySelector('img')).toBeNull();
    expect(root.querySelector('h3')?.textContent).toBe('User deleted successfully');
    expect(root.querySelector('pre')?.textContent).toContain(payload);
  });
});
